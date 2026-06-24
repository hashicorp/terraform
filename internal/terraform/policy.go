// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"
	"iter"
	"log"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func evaluatePolicies(ctx EvalContext, target addrs.AbsResourceInstance, config *configs.Resource, attrs, priorAttrs cty.Value, meta *proto.PolicyEvaluateResourceRequest_ResourceMetadata, callbacks callback.Functions) policy.EvaluationResponse {
	// We want a per-resource parent span so we can reason about the evaluation of individual
	// resources in the trace
	evalCtx := ctx.StopCtx()
	if phaseSpan := ctx.PolicyGraph().span; phaseSpan != nil {
		evalCtx = trace.ContextWithSpan(evalCtx, phaseSpan)
	}

	result := ctx.PolicyClient().EvaluateResource(evalCtx, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target:     target.Resource.Resource.Type,
		Attrs:      policy.CtyToPolicyValue(attrs),
		PriorAttrs: policy.CtyToPolicyValue(priorAttrs),
		Meta:       meta,
		Callbacks:  callbacks,
	})

	// Do a nil check because orphaned resources do not have a config, so we can't provide source information
	// for such errors.
	if config != nil {
		ptr := config.DeclRange.Ptr()
		for idx, diag := range result.Diagnostics {
			result.Diagnostics[idx] = diag.WithLocalRange(ptr)
		}
		for idx := range result.Enforcements {
			result.Enforcements[idx].LocalRange = ptr
		}
	}

	return result
}

func getResourcesForPolicyCallback(ctx EvalContext, walkOperation walkOperation, provider providers.Interface, schema providers.GetProviderSchemaResponse, config *configs.Config) func(callbackCtx context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
	return func(c context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
		_, span := tracer().Start(c, "policy.callback.getResources", trace.WithAttributes(
			attribute.String("policy.callback.getResources.type", target),
		))
		defer span.End()

		found := make([]cty.Value, 0)
		var filterMap map[string]cty.Value
		if !attrs.IsNull() {
			filterMap = attrs.AsValueMap()
		}
		var isPartialResult bool
		config.DeepEach(func(c *configs.Config) {
			state := ctx.State()
			for _, resource := range c.Module.ManagedResources {
				if resource.Type != target {
					continue
				}
				addr := resource.Addr().InModule(c.Path)
				schema := schema.SchemaForResourceAddr(addr.Resource)

				// Before checking the data to see if there is a match, check if there is a deferral for this address.
				//
				// If there is a deferral, we can't use the data to determine if there is a match so we'll indicate
				// the callback return is a partial result.
				deferred := ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{addr})
				if deferred {
					isPartialResult = true
					continue
				}

				// Now we implement a generator function that yields resource instances
				// from either the state or the config, depending on the walk operation.
				var resourcesSeq iter.Seq[cty.Value]
				if walkOperation == walkApply {
					// Read each config resource instance from the state, decoding it into a cty.Value
					resourcesSeq = states.ReadEachConfigResourceInstance(state, addr, func(inst *states.ResourceInstance) (cty.Value, bool) {
						if inst.Current == nil {
							return cty.NilVal, false
						}
						rsc, err := inst.Current.Decode(schema)
						if err != nil {
							log.Printf("[ERROR] getresources: failed to decode resource %q: %v", addr, err)
							return cty.NilVal, false
						}
						return rsc.Value, true
					})
				} else {
					// Read each config resource change from the plan, returning the corresponding cty.Value
					resourcesSeq = func(yield func(cty.Value) bool) {
						for change := range plans.ReadInstancesForConfigResource(ctx.Changes(), addr) {
							yield(change.After)
						}
					}
				}

				for resource := range resourcesSeq {
					matched, unknown := resourceMatchesFilter(addr, schema.Body, filterMap, resource)
					if matched {
						resource, _ = resource.UnmarkDeep()
						found = append(found, resource)
						continue
					}

					// If the filtered attribute for matching is unknown for this resource instance,
					// we can't determine whether it matches, so we'll mark the whole callback result as incomplete.
					// We still continue to the next resource instance, so that we return all known objects as well.
					isPartialResult = isPartialResult || unknown
				}
			}
		})
		span.SetAttributes(attribute.String("policy.callback.getResources.result_count", fmt.Sprintf("%d", len(found))))
		return found, isPartialResult, nil
	}
}

func getDataSourceForPolicyCallback(ctx EvalContext, provider providers.Interface, schema providers.GetProviderSchemaResponse, meta cty.Value) func(callbackCtx context.Context, datasource string, attrs cty.Value) (cty.Value, error) {
	return func(c context.Context, target string, attrs cty.Value) (cty.Value, error) {
		_, span := tracer().Start(c, "policy.callback.getDataSource", trace.WithAttributes(
			attribute.String("policy.callback.getDataSource.type", target),
		))
		defer span.End()
		if datasource, ok := schema.DataSources[target]; ok {
			configVal, err := datasource.Body.CoerceValue(attrs)
			if err != nil {
				return cty.NilVal, fmt.Errorf("invalid attributes for %q: %w", target, err)
			}

			validateResp := provider.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
				TypeName: target,
				Config:   configVal,
			})
			if err := validateResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, fmt.Errorf("failed to validate data source configuration: %s", err)
			}

			readResp := provider.ReadDataSource(providers.ReadDataSourceRequest{
				TypeName:     target,
				Config:       configVal,
				ProviderMeta: meta,
			})
			if err := readResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, fmt.Errorf("failed to read data source: %s", err)
			}

			return readResp.State, nil
		}
		return cty.NilVal, fmt.Errorf("no data source found for %s", target)
	}
}

// resourceMatchesFilter returns whether the given resource matches the given filter attributes and/or if the filter attributes are unknown for the resource.
func resourceMatchesFilter(addr addrs.ConfigResource, schema *configschema.Block, filterAttrs map[string]cty.Value, resource cty.Value) (matches, unknown bool) {
	if resource.IsNull() {
		// if the resource is null, then it doesn't match anything
		return false, false
	}
	if len(filterAttrs) == 0 {
		// if the filter is null, then match everything
		return true, false
	}

	sawUnknown := false

	attrTypes := resource.Type().AttributeTypes()
	for name, attr := range filterAttrs {
		if _, ok := attrTypes[name]; !ok {
			return false, false
		}

		equals := attr.Equals(resource.GetAttr(name))
		if !equals.IsKnown() {
			// If the filtered attribute for matching is unknown for this resource instance, we
			// can't determine whether it matches, so we track that we saw an unknown attribute and continue to check other attributes.
			// This lets false matches take precedence over the unknown result, since we can still determine that the resource does not match.
			sawUnknown = true
			continue
		}

		if equals.False() {
			log.Printf("[DEBUG] attribute %q does not match in resource %q", name, addr.String())
			// an attribute mismatch means we can't match this resource
			return false, false
		}
	}

	// We saw an unknown attribute, and no other attributes mismatched, so we can't determine whether the resource matches.
	if sawUnknown {
		return false, true
	}

	return true, false
}
