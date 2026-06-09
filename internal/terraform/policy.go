// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"iter"
	"log"
	"slices"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
)

func evaluatePolicies(ctx EvalContext, target addrs.AbsResourceInstance, config *configs.Resource, schema *configschema.Block, attrs, priorAttrs cty.Value, meta *proto.PolicyEvaluateResourceRequest_ResourceMetadata, callbacks callback.Functions) policy.EvaluationResponse {
	attrs, pvms := attrs.UnmarkDeepWithPaths()
	attrRedactedPaths, _ := marks.PathsWithMark(pvms, marks.Sensitive)
	priorAttrs, pvms = priorAttrs.UnmarkDeepWithPaths()
	priorAttrRedactedPaths, _ := marks.PathsWithMark(pvms, marks.Sensitive)

	result := ctx.PolicyClient().EvaluateResource(ctx.StopCtx(), policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: target.Resource.Resource.Type,
		Attrs: policy.PolicyValue{
			Raw:           attrs,
			RedactedPaths: attrRedactedPaths,
		},
		PriorAttrs: policy.PolicyValue{
			Raw:           priorAttrs,
			RedactedPaths: priorAttrRedactedPaths,
		},
		Meta:      meta,
		Callbacks: callbacks,
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
		var unknown bool
		config.DeepEach(func(c *configs.Config) {
			state := ctx.State()
			for _, resource := range c.Module.ManagedResources {
				if resource.Type != target {
					continue
				}
				addr := resource.Addr().InModule(c.Path)

				// Now we implement a generator function that yields resource instances
				// from either the state or the config, depending on the walk operation.
				var instanceSeq iter.Seq[cty.Value]
				if walkOperation == walkApply {
					instances := state.ResourceInstancesByConfig(addr)
					instanceSeq = func(yield func(cty.Value) bool) {
						for _, inst := range instances {
							if inst.Current == nil {
								continue
							}
							schema := schema.SchemaForResourceAddr(addr.Resource)
							rsc, err := inst.Current.Decode(schema)
							if err != nil {
								log.Printf("[ERROR] getresources: failed to decode resource %q: %v", addr, err)
								continue
							}
							if !yield(rsc.Value) {
								return
							}
						}
					}
				} else {
					changes := ctx.Changes().GetChangesForConfigResource(addr)
					instanceSeq = func(yield func(cty.Value) bool) {
						for _, change := range changes {
							if !yield(change.After) {
								return
							}
						}
					}
				}

				log.Printf("[DEBUG] getresources: found %d resources for policy target %q", len(slices.Collect(instanceSeq)), target)
				for resourceInstance := range instanceSeq {
					if attrs.IsNull() {
						// then match everything
						found = append(found, resourceInstance)
						continue
					}

					matched := true
					for name, attr := range attrs.AsValueMap() {
						if !resourceInstance.Type().HasAttribute(name) {
							log.Printf("[DEBUG] attribute %q not found in resource %q", name, addr.String())
							matched = false
							break
						}

						instanceAttr := resourceInstance.GetAttr(name)
						equals := attr.Equals(instanceAttr)
						if !equals.IsKnown() {
							// The filtered attribute for matching is unknown for this resource instance,
							// so we can't determine whether it matches. We'll mark the callback result as incomplete.
							// We still continue to the next resource instance, so that we return all known objects as well.
							unknown = true
							matched = false
							continue
						}

						if equals.False() {
							matched = false
							log.Printf("[DEBUG] attribute %q does not match in resource %q", name, addr.String())
							continue
						}
					}

					if matched {
						resourceInstance, _ = resourceInstance.UnmarkDeep()
						found = append(found, resourceInstance)
					}

				}
			}
		})
		return found, unknown, nil
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
