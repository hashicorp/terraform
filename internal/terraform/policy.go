// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
)

func evaluatePolicies(ctx EvalContext, target addrs.AbsResourceInstance, config *configs.Resource, schema *configschema.Block, attrs, priorAttrs cty.Value, meta *proto.ResourceMetadata, callbacks callback.Functions) policy.EvaluationResponse {
	var attrRedactedPaths []cty.Path
	var priorAttrRedactedPaths []cty.Path
	if schema != nil {
		attrRedactedPaths = schema.SensitivePaths(attrs, nil)
		priorAttrRedactedPaths = schema.SensitivePaths(priorAttrs, nil)
	}

	result := ctx.PolicyClient().Evaluate(ctx.StopCtx(), policy.EvaluationRequest[*proto.ResourceMetadata]{
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

	// orphaned resources do not have a config, so we can't provide source information
	// for these errors.
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

func getResourcesForPolicyCallback(ctx EvalContext, config *configs.Config) func(target string, attrs cty.Value) ([]cty.Value, error) {
	return func(target string, attrs cty.Value) ([]cty.Value, error) {
		var found []cty.Value
		config.DeepEach(func(c *configs.Config) {
			for _, resource := range c.Module.ManagedResources {
				if resource.Type != target {
					continue
				}

				resources := ctx.Changes().GetChangesForConfigResource(resource.Addr().InModule(c.Path))
				for _, change := range resources {
					resource := change.After
					if attrs.IsNull() {
						// then match everything
						found = append(found, resource)
						continue
					}

					value, matched := resource, true
					for name, attr := range attrs.AsValueMap() {
						if !value.Type().HasAttribute(name) {
							matched = false
							break
						}

						equals := attr.Equals(value.GetAttr(name))
						if !equals.IsKnown() {
							// We'll treat unknown values as matches, and they
							// can be handled on the Terraform Policy side.
							continue
						}

						if equals.False() {
							matched = false
							break
						}
					}

					if matched {
						value, _ = value.UnmarkDeep()
						found = append(found, value)
					}

				}
			}
		})
		return found, nil
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
