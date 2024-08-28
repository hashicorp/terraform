// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"

	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	TestingResourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":    {Type: cty.String, Optional: true, Computed: true},
			"value": {Type: cty.String, Optional: true},
		},
	}

	DeferredResourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":       {Type: cty.String, Optional: true, Computed: true},
			"value":    {Type: cty.String, Optional: true},
			"deferred": {Type: cty.Bool, Required: true},
		},
	}

	FailedResourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":         {Type: cty.String, Optional: true, Computed: true},
			"value":      {Type: cty.String, Optional: true},
			"fail_plan":  {Type: cty.Bool, Optional: true, Computed: true},
			"fail_apply": {Type: cty.Bool, Optional: true, Computed: true},
		},
	}

	TestingDataSourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":    {Type: cty.String, Required: true},
			"value": {Type: cty.String, Computed: true},
		},
	}
)

// MockProvider wraps the standard MockProvider with a simple in-memory
// data store for resources and data sources.
type MockProvider struct {
	*testing_provider.MockProvider

	ResourceStore *ResourceStore
}

// NewProvider returns a new MockProvider with an empty data store.
func NewProvider() *MockProvider {
	return NewProviderWithData(NewResourceStore())
}

// NewProviderWithData returns a new MockProvider with the given data store.
func NewProviderWithData(store *ResourceStore) *MockProvider {
	if store == nil {
		store = NewResourceStore()
	}

	return &MockProvider{
		MockProvider: &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"configure_error": {
								Type:     cty.String,
								Optional: true,
							},
							"ignored": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
				ResourceTypes: map[string]providers.Schema{
					"testing_resource": {
						Block: TestingResourceSchema,
					},
					"testing_deferred_resource": {
						Block: DeferredResourceSchema,
					},
					"testing_failed_resource": {
						Block: FailedResourceSchema,
					},
				},
				DataSources: map[string]providers.Schema{
					"testing_data_source": {
						Block: TestingDataSourceSchema,
					},
				},
				Functions: map[string]providers.FunctionDecl{
					"echo": {
						Parameters: []providers.FunctionParam{
							{Name: "value", Type: cty.DynamicPseudoType},
						},
						ReturnType: cty.DynamicPseudoType,
					},
				},
				ServerCapabilities: providers.ServerCapabilities{
					MoveResourceState: true,
				},
			},
			ConfigureProviderFn: func(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
				// If configure_error is set, return an error.
				err := request.Config.GetAttr("configure_error")
				if err.IsKnown() && !err.IsNull() {
					return providers.ConfigureProviderResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.AttributeValue(tfdiags.Error, err.AsString(), "configure_error attribute was set", cty.GetAttrPath("configure_error")),
						},
					}
				}
				return providers.ConfigureProviderResponse{}
			},
			PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				if request.ProposedNewState.IsNull() {
					// Deleting, so just return the proposed change.
					return providers.PlanResourceChangeResponse{
						PlannedState: request.ProposedNewState,
					}
				}

				// We're creating or updating, so we need to return the new
				// state with any computed values filled in.

				value := request.ProposedNewState
				if id := value.GetAttr("id"); id.IsNull() {
					vals := value.AsValueMap()
					vals["id"] = cty.UnknownVal(cty.String)
					value = cty.ObjectVal(vals)
				}

				if request.TypeName == "testing_deferred_resource" {
					if value.GetAttr("deferred").True() {
						return providers.PlanResourceChangeResponse{
							PlannedState: value,
							Deferred: &providers.Deferred{
								Reason: providers.DeferredReasonResourceConfigUnknown,
							},
						}
					}
				}

				if request.TypeName == "testing_failed_resource" {
					// First, populate the fail attributes with null if they are
					// null
					if value.GetAttr("fail_apply").IsNull() {
						vals := value.AsValueMap()
						vals["fail_apply"] = cty.False
						value = cty.ObjectVal(vals)
					}
					if value.GetAttr("fail_plan").IsNull() {
						vals := value.AsValueMap()
						vals["fail_plan"] = cty.False
						value = cty.ObjectVal(vals)
					}

					// If fail_plan is set, return a planned failure.
					if value.GetAttr("fail_plan").True() {
						return providers.PlanResourceChangeResponse{
							PlannedState: value,
							Diagnostics: tfdiags.Diagnostics{
								tfdiags.Sourceless(tfdiags.Error, "planned failure", "plan failure"),
							},
						}
					}
				}

				return providers.PlanResourceChangeResponse{
					PlannedState: value,
				}
			},
			ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
				if request.PlannedState.IsNull() {
					// Deleting, so just update the store and return.
					store.Delete(request.PriorState.GetAttr("id").AsString())
					return providers.ApplyResourceChangeResponse{
						NewState: request.PlannedState,
					}
				}

				// Creating or updating, so update the store and return.

				// First, populate the computed value if we have to.
				value := request.PlannedState
				if id := value.GetAttr("id"); !id.IsKnown() {
					vals := value.AsValueMap()
					vals["id"] = cty.StringVal(mustGenerateUUID())
					value = cty.ObjectVal(vals)
				}

				if request.TypeName == "testing_failed_resource" {
					if value.GetAttr("fail_apply").True() {
						return providers.ApplyResourceChangeResponse{
							Diagnostics: tfdiags.Diagnostics{
								tfdiags.Sourceless(tfdiags.Error, "planned failure", "apply failure"),
							},
						}
					}
				}

				// Finally, update the store and return.
				store.Set(value.GetAttr("id").AsString(), value)
				return providers.ApplyResourceChangeResponse{
					NewState: value,
				}
			},
			ReadResourceFn: func(request providers.ReadResourceRequest) providers.ReadResourceResponse {
				var diags tfdiags.Diagnostics

				id := request.PriorState.GetAttr("id").AsString()
				value, exists := store.Get(id)
				if !exists {
					// Then we'll just behave as if the resource was destroyed
					// externally.
					switch request.TypeName {
					case "testing_failed_resource":
						value = cty.NullVal(FailedResourceSchema.ImpliedType())
					case "testing_deferred_resource":
						value = cty.NullVal(DeferredResourceSchema.ImpliedType())
					case "testing_resource":
						value = cty.NullVal(TestingResourceSchema.ImpliedType())
					default:
						panic(fmt.Sprintf("unknown resource type %q", request.TypeName))
					}
				}
				return providers.ReadResourceResponse{
					NewState:    value,
					Diagnostics: diags,
				}
			},
			ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
				var diags tfdiags.Diagnostics

				id := request.Config.GetAttr("id").AsString()
				value, exists := store.Get(id)
				if !exists {
					diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%q not found", id)))
				}
				return providers.ReadDataSourceResponse{
					State:       value,
					Diagnostics: diags,
				}
			},
			ImportResourceStateFn: func(request providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
				id := request.ID
				value, exists := store.Get(id)
				if !exists {
					return providers.ImportResourceStateResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Error, "not found", fmt.Sprintf("%q not found", id)),
						},
					}
				}
				return providers.ImportResourceStateResponse{
					ImportedResources: []providers.ImportedResource{
						{
							TypeName: request.TypeName,
							State:    value,
						},
					},
				}
			},
			MoveResourceStateFn: func(request providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
				if request.SourceTypeName != "testing_resource" && request.TargetTypeName != "testing_deferred_resource" {
					return providers.MoveResourceStateResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Error, "unsupported", "unsupported move"),
						},
					}
				}
				// So, we know we're moving from `testing_resource` to
				// `testing_deferred_resource`.

				source, err := ctyjson.Unmarshal(request.SourceStateJSON, cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))
				if err != nil {
					return providers.MoveResourceStateResponse{
						Diagnostics: tfdiags.Diagnostics{
							tfdiags.Sourceless(tfdiags.Error, "invalid source state", err.Error()),
						},
					}
				}

				target := cty.ObjectVal(map[string]cty.Value{
					"id":       source.GetAttr("id"),
					"value":    source.GetAttr("value"),
					"deferred": cty.False,
				})
				store.Set(source.GetAttr("id").AsString(), target)

				return providers.MoveResourceStateResponse{
					TargetState: target,
				}
			},
			CallFunctionFn: func(request providers.CallFunctionRequest) providers.CallFunctionResponse {
				// Just echo the first argument back as the result.
				return providers.CallFunctionResponse{
					Result: request.Arguments[0],
				}
			},
		},
		ResourceStore: store,
	}
}

// mustGenerateUUID is a helper to generate a UUID and panic if it fails.
func mustGenerateUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}
