// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"
	"runtime/debug"
	"testing"

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

	BlockedResourceSchema = &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":                 {Type: cty.String, Optional: true, Computed: true},
			"value":              {Type: cty.String, Optional: true},
			"required_resources": {Type: cty.Set(cty.String), Optional: true},
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

	// If set, authentication means the configuration must provide a value
	// that matches the value here otherwise the Configure function will
	// fail.
	Authentication string
}

// NewProvider returns a new MockProvider with an empty data store.
func NewProvider(t *testing.T) *MockProvider {
	provider := NewProviderWithData(t, NewResourceStore())
	return provider
}

// NewProviderWithData returns a new MockProvider with the given data store.
func NewProviderWithData(t *testing.T, store *ResourceStore) *MockProvider {
	if store == nil {
		store = NewResourceStore()
	}

	// grab the current stack trace so we know where the provider was created
	// in case it isn't being cleaned up properly
	currentStackTrace := debug.Stack()

	provider := &MockProvider{
		MockProvider: &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							// if the configuration sets require_auth then it
							// must also provide the correct value for
							// authentication
							"authentication": {
								Type:      cty.String,
								Sensitive: true,
								Optional:  true,
							},
							"require_auth": {
								Type:     cty.Bool,
								Optional: true,
							},

							// If this value is provider, the Configure
							// function call will fail and return the value
							// here as part of the error.
							"configure_error": {
								Type:     cty.String,
								Optional: true,
							},

							// ignored allows the configuration to create
							// dependencies from this provider to component
							// blocks and inputs without affecting behaviour.
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
					"testing_blocked_resource": {
						Block: BlockedResourceSchema,
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
			PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				return getResource(request.TypeName).Plan(request, store)
			},
			ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
				return getResource(request.TypeName).Apply(request, store)
			},
			ReadResourceFn: func(request providers.ReadResourceRequest) providers.ReadResourceResponse {
				return getResource(request.TypeName).Read(request, store)
			},
			ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
				var diags tfdiags.Diagnostics

				id := request.Config.GetAttr("id")
				if id.IsNull() {
					diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "missing id", "id is required"))
					return providers.ReadDataSourceResponse{
						Diagnostics: diags,
					}
				}

				value, exists := store.Get(id.AsString())
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

	// We want to use internal fields in this function so we have to set it
	// like this.
	provider.ConfigureProviderFn = provider.configure

	t.Cleanup(func() {
		// Fail the test if this provider is not closed.
		if !provider.CloseCalled {
			t.Log(string(currentStackTrace))
			t.Fatalf("provider.Close was not called")
		}
	})

	return provider
}

func (provider *MockProvider) configure(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// If configure_error is set, return an error.
	err := request.Config.GetAttr("configure_error")
	if err.IsKnown() && !err.IsNull() {
		return providers.ConfigureProviderResponse{
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.AttributeValue(tfdiags.Error, err.AsString(), "configure_error attribute was set", cty.GetAttrPath("configure_error")),
			},
		}
	}

	// We deliberately only check the authentication if the configuration
	// is providing it. It's entirely up to the config to opt into the
	// authentication which would be crazy for a real provider but just
	// makes things so much simpler for us in testing world.
	requireAuth := request.Config.GetAttr("require_auth")
	if requireAuth.True() {
		authn := request.Config.GetAttr("authentication")
		if authn.IsNull() || !authn.IsKnown() {
			return providers.ConfigureProviderResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.AttributeValue(tfdiags.Error, "Authentication failed", "authentication field is required", cty.GetAttrPath("authentication")),
				},
			}
		}
		if authn.AsString() != provider.Authentication {
			return providers.ConfigureProviderResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.AttributeValue(tfdiags.Error, "Authentication failed", "authentication field did not match expected", cty.GetAttrPath("authentication")),
				},
			}
		}
	}

	return providers.ConfigureProviderResponse{}
}

// mustGenerateUUID is a helper to generate a UUID and panic if it fails.
func mustGenerateUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}
