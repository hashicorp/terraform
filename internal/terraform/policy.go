// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"iter"
	"log"

	"github.com/zclconf/go-cty/cty"

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
	result := ctx.PolicyClient().EvaluateResource(ctx.StopCtx(), policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
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

func getResourcesForPolicyCallback(ctx EvalContext, walkOperation walkOperation, provider providers.Interface, schema providers.GetProviderSchemaResponse, config *configs.Config) func(target string, attrs cty.Value) ([]cty.Value, bool, error) {
	return func(target string, attrs cty.Value) ([]cty.Value, bool, error) {
		var found []cty.Value
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

				// Now we implement a generator function that yields resource instances
				// from either the state or the config, depending on the walk operation.
				var resourcesSeq iter.Seq[cty.Value]
				var count int
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
						count++
						return rsc.Value, true
					})
				} else {
					// Read each config resource change from the plan, returning the corresponding cty.Value
					resourcesSeq = func(yield func(cty.Value) bool) {
						for change := range plans.ReadInstancesForConfigResource(ctx.Changes(), addr) {
							count++
							yield(change.After)
						}
					}
				}

				for resource := range resourcesSeq {
					matched, unknown := resourceMatchesFilter(addr, schema.Body, filterMap, resource)
					if matched {
						resource, _ = resource.UnmarkDeep()
						found = append(found, resource)
					}

					// The filtered attribute for matching is unknown for this resource instance,
					// so we can't determine whether it matches. We'll mark the whole callback result as incomplete.
					// We still continue to the next resource instance, so that we return all known objects as well.
					isPartialResult = isPartialResult || unknown

				}
			}
		})
		return found, isPartialResult, nil
	}
}

func getDataSourceForPolicyCallback(ctx EvalContext, provider providers.Interface, schema providers.GetProviderSchemaResponse, meta cty.Value) func(datasource string, attrs cty.Value) (cty.Value, error) {
	return func(target string, attrs cty.Value) (cty.Value, error) {
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

	for name, attr := range filterAttrs {
		if !resource.Type().HasAttribute(name) {
			log.Printf("[DEBUG] attribute %q not found in resource %q", name, addr.String())
			return false, false
		}

		if schema == nil || schema.Attributes[name] == nil {
			return false, false
		}

		equals := attr.Equals(resource.GetAttr(name))
		if !equals.IsKnown() {
			// A filtered attribute for matching is unknown for this resource instance.
			// We can't determine whether it matches. We track that we saw an unknown attribute, but continue to check other attributes.
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
