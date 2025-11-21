// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluggable

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/zclconf/go-cty/cty"
)

func TestNewPluggable(t *testing.T) {
	cases := map[string]struct {
		provider providers.Interface
		typeName string

		wantError string
	}{
		"no error when inputs are provided": {
			provider: &testing_provider.MockProvider{},
			typeName: "foo_bar",
		},
		"no error when store name has underscores": {
			provider: &testing_provider.MockProvider{},
			// foo provider containing fizz_buzz store
			typeName: "foo_fizz_buzz",
		},
		"error when store type not provided": {
			provider:  &testing_provider.MockProvider{},
			typeName:  "",
			wantError: "Attempted to initialize pluggable state with an empty string identifier for the state store.",
		},
		"error when provider interface is nil": {
			provider:  nil,
			typeName:  "foo_bar",
			wantError: "Attempted to initialize pluggable state with a nil provider interface.",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			_, err := NewPluggable(tc.provider, tc.typeName)
			if err != nil {
				if tc.wantError == "" {
					t.Fatalf("unexpected error: %s", err)
				}
				if !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error %q but got %q", tc.wantError, err)
				}
				return
			}
			if err == nil && tc.wantError != "" {
				t.Fatalf("expected error %q but got none", tc.wantError)
			}
		})
	}
}

func TestPluggable_ConfigSchema(t *testing.T) {

	p := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider:          providers.Schema{},
			DataSources:       map[string]providers.Schema{},
			ResourceTypes:     map[string]providers.Schema{},
			ListResourceTypes: map[string]providers.Schema{},
			StateStores: map[string]providers.Schema{
				// This imagines a provider called foo that contains
				// two pluggable state store implementations, called
				// bar and baz.
				// It's accurate to include the prefixed provider name
				// in the keys of schema maps
				"foo_bar": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"bar": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
				"foo_baz": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"baz": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}

	cases := map[string]struct {
		provider providers.Interface
		typeName string

		expectedAttrName string
		expectNil        bool
	}{
		"returns expected schema - bar store": {
			provider:         p,
			typeName:         "foo_bar",
			expectedAttrName: "bar",
		},
		"returns expected schema - baz store": {
			provider:         p,
			typeName:         "foo_baz",
			expectedAttrName: "baz",
		},
		"returns nil if there isn't a store with a matching name": {
			provider:  p,
			typeName:  "foo_not_implemented",
			expectNil: true,
		},
		"returns nil if no state stores are implemented in the provider": {
			provider:  &testing_provider.MockProvider{},
			typeName:  "foo_bar",
			expectNil: true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			p, err := NewPluggable(tc.provider, tc.typeName)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			s := p.ConfigSchema()
			if mock, ok := tc.provider.(*testing_provider.MockProvider); ok {
				if !mock.GetProviderSchemaCalled {
					t.Fatal("expected mock's GetProviderSchema method to have been called")
				}
			}
			if s == nil {
				if !tc.expectNil {
					t.Fatal("ConfigSchema returned an unexpected nil schema")
				}
				return
			}
			if val := s.Attributes[tc.expectedAttrName]; val == nil {
				t.Fatalf("expected the returned schema to include an attr called %q, but it was missing. Schema contains attrs: %v",
					tc.expectedAttrName,
					slices.Sorted(maps.Keys(s.Attributes)))
			}
		})
	}
}

func TestPluggable_PrepareConfig(t *testing.T) {
	fooBar := "foo_bar"
	cases := map[string]struct {
		provider providers.Interface
		typeName string
		config   cty.Value

		wantError string
	}{
		"when config is deemed valid there are no diagnostics": {
			provider: &testing_provider.MockProvider{
				ConfigureProviderCalled: true,
				ValidateStateStoreConfigFn: func(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
					// if validation is ok, response has no diags
					return providers.ValidateStateStoreConfigResponse{}
				},
			},
			typeName: fooBar,
		},
		"errors are returned, and expected arguments are in the request": {
			provider: &testing_provider.MockProvider{
				ConfigureProviderCalled: true,
				ValidateStateStoreConfigFn: func(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
					// Are the right values being put into the incoming request?
					if req.TypeName != fooBar || req.Config != cty.True {
						t.Fatalf("expected provider ValidateStateStoreConfig method to receive TypeName %q and Config %q, instead got TypeName %q and Config %q",
							fooBar,
							cty.True,
							req.TypeName,
							req.Config)
					}

					// Force an error, to see it makes it back to the invoked method ok
					resp := providers.ValidateStateStoreConfigResponse{}
					resp.Diagnostics = resp.Diagnostics.Append(errors.New("error diagnostic raised from mock"))
					return resp
				},
			},
			typeName:  fooBar,
			config:    cty.BoolVal(true),
			wantError: "error diagnostic raised from mock",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			p, err := NewPluggable(tc.provider, tc.typeName)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			_, diags := p.PrepareConfig(tc.config)
			if mock, ok := tc.provider.(*testing_provider.MockProvider); ok {
				if !mock.ValidateStateStoreConfigCalled {
					t.Fatal("expected mock's ValidateStateStoreConfig method to have been called")
				}
			}
			if diags.HasErrors() {
				if tc.wantError == "" {
					t.Fatalf("unexpected error: %s", diags.Err())
				}
				if !strings.Contains(diags.Err().Error(), tc.wantError) {
					t.Fatalf("expected error %q but got: %q", tc.wantError, diags.Err())
				}
				return
			}
			if !diags.HasErrors() && tc.wantError != "" {
				t.Fatal("expected an error but got none")
			}
		})
	}
}

func TestPluggable_Configure(t *testing.T) {

	// Arrange mocks
	typeName := "foo_bar"
	wantError := "error diagnostic raised from mock"
	mock := &testing_provider.MockProvider{
		ConfigureProviderCalled:        true,
		ValidateStateStoreConfigCalled: true,
		ConfigureStateStoreFn: func(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
			if req.TypeName != typeName || req.Config != cty.True {
				t.Fatalf("expected provider ConfigureStateStore method to receive TypeName %q and Config %q, instead got TypeName %q and Config %q",
					typeName,
					cty.True,
					req.TypeName,
					req.Config)
			}

			resp := providers.ConfigureStateStoreResponse{}
			resp.Diagnostics = resp.Diagnostics.Append(errors.New(wantError))
			return resp
		},
	}

	// Make Pluggable and invoke Configure
	p, err := NewPluggable(mock, typeName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// This isn't representative of true values used with the method, but is sufficient
	// for testing that the mock receives the provided value as expected.
	config := cty.BoolVal(true)
	diags := p.Configure(config)

	// Assertions
	if !mock.ValidateStateStoreConfigCalled {
		t.Fatal("expected mock's ValidateStateStoreConfig method to have been called")
	}
	if !diags.HasErrors() {
		t.Fatal("expected an error but got none")
	}
	if !strings.Contains(diags.Err().Error(), wantError) {
		t.Fatalf("expected error %q but got: %q", wantError, diags.Err())
	}
}

func TestPluggable_Workspaces(t *testing.T) {
	fooBar := "foo_bar"
	cases := map[string]struct {
		provider           providers.Interface
		expectedWorkspaces []string
		wantError          string
	}{
		"returned workspaces match what's returned from the store": {
			// and "default" isn't included by default
			provider: &testing_provider.MockProvider{
				ConfigureProviderCalled:        true,
				ValidateStateStoreConfigCalled: true,
				ConfigureStateStoreCalled:      true,
				GetStatesFn: func(req providers.GetStatesRequest) providers.GetStatesResponse {
					workspaces := []string{"abcd", "efg"}
					resp := providers.GetStatesResponse{
						States: workspaces,
					}
					return resp
				},
			},
			expectedWorkspaces: []string{"abcd", "efg"},
		},
		"errors are returned, and expected arguments are in the request": {
			provider: &testing_provider.MockProvider{
				ConfigureProviderCalled:        true,
				ValidateStateStoreConfigCalled: true,
				ConfigureStateStoreCalled:      true,
				GetStatesFn: func(req providers.GetStatesRequest) providers.GetStatesResponse {
					if req.TypeName != fooBar {
						t.Fatalf("expected provider GetStates method to receive TypeName %q, instead got TypeName %q",
							fooBar,
							req.TypeName)
					}
					resp := providers.GetStatesResponse{}
					resp.Diagnostics = resp.Diagnostics.Append(errors.New("error diagnostic raised from mock"))
					return resp
				},
			},
			wantError: "error diagnostic raised from mock",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			p, err := NewPluggable(tc.provider, fooBar)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			workspaces, wDiags := p.Workspaces()
			if mock, ok := tc.provider.(*testing_provider.MockProvider); ok {
				if !mock.GetStatesCalled {
					t.Fatal("expected mock's GetStates method to have been called")
				}
			}
			if wDiags.HasErrors() {
				if tc.wantError == "" {
					t.Fatalf("unexpected error: %s", err)
				}
				if !strings.Contains(wDiags.Err().Error(), tc.wantError) {
					t.Fatalf("expected error %q but got: %q", tc.wantError, err)
				}
				return
			}

			if tc.wantError != "" {
				t.Fatal("expected an error but got none")
			}

			if slices.Compare(workspaces, tc.expectedWorkspaces) != 0 {
				t.Fatalf("expected workspaces %v, got %v", tc.expectedWorkspaces, workspaces)
			}
		})
	}
}

func TestPluggable_DeleteWorkspace(t *testing.T) {

	// Arrange mocks
	typeName := "foo_bar"
	stateId := "my-state"
	mock := &testing_provider.MockProvider{
		ConfigureProviderCalled:        true,
		ValidateStateStoreConfigCalled: true,
		ConfigureStateStoreCalled:      true,
		DeleteStateFn: func(req providers.DeleteStateRequest) providers.DeleteStateResponse {
			if req.TypeName != typeName || req.StateId != stateId {
				t.Fatalf("expected provider DeleteState method to receive TypeName %q and StateId %q, instead got TypeName %q and StateId %q",
					typeName,
					stateId,
					req.TypeName,
					req.StateId,
				)
			}
			resp := providers.DeleteStateResponse{}
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("error diagnostic raised from mock"))
			return resp
		},
	}

	// Make Pluggable and invoke DeleteWorkspace
	p, err := NewPluggable(mock, typeName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	dwDiags := p.DeleteWorkspace(stateId, false)

	// Assertions
	if !mock.DeleteStateCalled {
		t.Fatal("expected mock's DeleteState method to have been called")
	}

	if !dwDiags.HasErrors() {
		t.Fatal("test is expected to return an error, but there isn't one")
	}
	wantError := "error diagnostic raised from mock"
	if !strings.Contains(dwDiags.Err().Error(), wantError) {
		t.Fatalf("expected error %q but got: %q", wantError, err)
	}
}

func TestPluggable_ProviderSchema(t *testing.T) {
	t.Run("Returns the expected provider schema", func(t *testing.T) {
		mock := &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"custom_attr": {Type: cty.String, Optional: true},
						},
					},
				},
			},
		}
		p, err := NewPluggable(mock, "foobar")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		providerSchema := p.ProviderSchema()

		if !mock.GetProviderSchemaCalled {
			t.Fatal("expected ProviderSchema to call the GetProviderSchema RPC")
		}
		if providerSchema == nil {
			t.Fatal("ProviderSchema returned an unexpected nil schema")
		}
		if val := providerSchema.Attributes["custom_attr"]; val == nil {
			t.Fatalf("expected the returned schema to include an attr called %q, but it was missing. Schema contains attrs: %v",
				"custom_attr",
				slices.Sorted(maps.Keys(providerSchema.Attributes)))
		}
	})

	t.Run("Returns a nil schema when the provider has an empty schema", func(t *testing.T) {
		mock := &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				Provider: providers.Schema{
					// empty schema
				},
			},
		}
		p, err := NewPluggable(mock, "foobar")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		providerSchema := p.ProviderSchema()

		if !mock.GetProviderSchemaCalled {
			t.Fatal("expected ProviderSchema to call the GetProviderSchema RPC")
		}
		if providerSchema != nil {
			t.Fatalf("expected ProviderSchema to return a nil schema but got: %#v", providerSchema)
		}
	})
}
