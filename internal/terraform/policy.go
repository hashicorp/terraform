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

	"github.com/hashicorp/hcl/v2"
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
	if pg := ctx.PolicyGraph(); pg != nil {
		if phaseSpan := pg.span; phaseSpan != nil {
			evalCtx = trace.ContextWithSpan(evalCtx, phaseSpan)
		}
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
		result = result.WithLocalRange(config.DeclRange.Ptr())
	}

	return result
}

func (cb *PolicyCallbackManager) GetResourcesCallback(ctx EvalContext, provider providers.Interface) func(callbackCtx context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
	return func(c context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
		_, span := tracer().Start(c, "policy.callback.getResources", trace.WithAttributes(
			attribute.String("policy.callback.getResources.type", target),
		))
		defer span.End()

		var filterMap map[string]cty.Value
		if !attrs.IsNull() {
			filterMap = attrs.AsValueMap()
		}

		found := make([]cty.Value, 0)
		candidates, isPartialResult := collectPolicyCandidates(ctx, cb.WalkOperation, cb.Schema, cb.Config, target)
		for _, cand := range candidates {
			matched, unknown := resourceMatchesFilter(cand.addr, nil, filterMap, cand.value)
			if matched {
				resource, _ := cand.value.UnmarkDeep()
				found = append(found, resource)
				continue
			}
			// If the filtered attribute for matching is unknown for this resource
			// instance, we can't determine whether it matches, so we mark the
			// whole callback result as incomplete. We still return known objects.
			isPartialResult = isPartialResult || unknown
		}
		span.SetAttributes(attribute.String("policy.callback.getResources.result_count", fmt.Sprintf("%d", len(found))))
		return found, isPartialResult, nil
	}
}

// policyCandidate is a single candidate target resource instance collected for a
// relationship/getresources lookup, paired with its address for diagnostics and
// its config body for reference (provenance) verification.
type policyCandidate struct {
	addr   addrs.ConfigResource
	value  cty.Value
	config hcl.Body
}

// collectPolicyCandidates walks the configuration and returns every resource
// instance of the requested type (from state during apply, from planned changes
// otherwise), skipping addresses whose dependencies are deferred. The returned
// bool reports whether any deferral was encountered, which makes the callback
// result partial. This is the single iteration path shared by both the linear
// matcher and the connector index so their semantics can't drift.
func collectPolicyCandidates(ctx EvalContext, walkOperation walkOperation, schema providers.GetProviderSchemaResponse, config *configs.Config, target string) ([]policyCandidate, bool) {
	candidates := make([]policyCandidate, 0)
	var isPartialResult bool
	config.DeepEach(func(c *configs.Config) {
		state := ctx.State()
		for _, resource := range c.Module.ManagedResources {
			if resource.Type != target {
				continue
			}
			addr := resource.Addr().InModule(c.Path)
			schema := schema.SchemaForResourceAddr(addr.Resource)
			cfgBody := resource.Config

			// Skip addresses with deferred dependencies: we can't use their data
			// to determine a match, so the result is partial.
			if ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{addr}) {
				isPartialResult = true
				continue
			}

			var resourcesSeq iter.Seq[cty.Value]
			if walkOperation == walkApply {
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
				resourcesSeq = func(yield func(cty.Value) bool) {
					for change := range plans.ReadInstancesForConfigResource(ctx.Changes(), addr) {
						yield(change.After)
					}
				}
			}

			for resource := range resourcesSeq {
				candidates = append(candidates, policyCandidate{addr: addr, value: resource, config: cfgBody})
			}
		}
	})
	return candidates, isPartialResult
}

func (cb *PolicyCallbackManager) GetDataSourceCallback(ctx EvalContext, provider providers.Interface) func(callbackCtx context.Context, datasource string, attrs cty.Value) (cty.Value, bool, error) {
	return func(c context.Context, target string, attrs cty.Value) (cty.Value, bool, error) {
		_, span := tracer().Start(c, "policy.callback.getDataSource", trace.WithAttributes(
			attribute.String("policy.callback.getDataSource.type", target),
		))
		defer span.End()
		schema := cb.Schema
		if datasource, ok := schema.DataSources[target]; ok {
			configVal, err := datasource.Body.CoerceValue(attrs)
			if err != nil {
				return cty.NilVal, false, fmt.Errorf("invalid attributes for %q: %w", target, err)
			}

			validateResp := provider.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
				TypeName: target,
				Config:   configVal,
			})
			if err := validateResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, false, fmt.Errorf("failed to validate data source configuration: %s", err)
			}

			meta := cty.NilVal
			if schema.ProviderMeta.Body != nil {
				meta = cty.NullVal(schema.ProviderMeta.Body.ImpliedType())
			}

			readResp := provider.ReadDataSource(providers.ReadDataSourceRequest{
				TypeName:           target,
				Config:             configVal,
				ClientCapabilities: ctx.ClientCapabilities(),
				ProviderMeta:       meta,
			})
			if err := readResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, false, fmt.Errorf("failed to read data source: %s", err)
			}

			// If the data source indicates deferral (which would be unlikely here), we need to pass that info back to the caller
			deferred := false
			if readResp.Deferred != nil {
				// If we don't support deferrals, but the provider reports a deferral, we should emit an error.
				if !ctx.Deferrals().DeferralAllowed() {
					return cty.NilVal, false, fmt.Errorf("failed to read data source: "+
						"The provider signaled a deferred action for %s, but in this context deferrals are disabled. "+
						"This is a bug in the provider, please file an issue with the provider developers.", target)
				}

				deferred = true
			}

			return readResp.State, deferred, nil
		}
		return cty.NilVal, false, fmt.Errorf("no data source found for %s", target)
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
