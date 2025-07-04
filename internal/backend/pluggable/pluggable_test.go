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
		"error when provider interface is nil ": {
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
						t.Fatalf("expected provider ValidateStateStoreConfig method to receive typeName %q and config %q, instead got typeName %q and config %q",
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
