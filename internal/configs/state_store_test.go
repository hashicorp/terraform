// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// The Hash method assumes that the state_store schema doesn't include a provider block,
// and it requires calling code to remove the nested provider block from state_store config data.
func TestStateStore_Hash(t *testing.T) {

	// This test assumes a configuration like this,
	// where the "fs" state store is implemented in
	// the "foobar" provider:
	//
	// terraform {
	//   required_providers = {
	//     # entries would be here
	//   }
	//   state_store "foobar_fs" {
	//     # Nested provider block
	//     provider "foobar" {
	//       foobar = "foobar"
	//     }

	//     # Attributes for configuring the state store
	//     path          = "mystate.tfstate"
	//     workspace_dir = "foobar"
	//   }
	// }

	// Normally these schemas would come from a provider's GetProviderSchema data
	stateStoreSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"path": {
				Type:     cty.String,
				Required: true,
			},
			"workspace_dir": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}
	providerSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foobar": {
				Type:     cty.String,
				Required: true,
			},
		},
	}

	cases := map[string]struct {
		config          hcl.Body
		providerConfig  hcl.Body
		schema          *configschema.Block
		wantErrorString string
	}{
		"ignores the provider block in config data, as long as the schema doesn't include it": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			providerConfig: configBodyForTest(t, `foobar = "foobar"`),
		},
		"tolerates empty config block for the provider even when schema has Required field(s)": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `
					provider "foobar" {
						# required field "foobar" is missing
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			providerConfig: hcl.EmptyBody(),
		},
		"tolerates missing Required field(s) in state_store config": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `
					provider "foobar" {
					  foobar = "foobar"
					}
					
					# required field "path" is missing
					workspace_dir = "foobar"`),
			providerConfig: hcl.EmptyBody(),
		},
		"returns errors when the config contains non-provider things that aren't in the schema": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `
					provider "foobar" {
					  foobar = "foobar"
					}
					unexpected_block {
					  foobar = "foobar"
					}
					unexpected_attr = "foobar"
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			providerConfig:  configBodyForTest(t, `foobar = "foobar"`),
			wantErrorString: "Unsupported argument",
		},
		"returns an error if the schema includes a provider block": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"provider": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
			},
			config: configBodyForTest(t, `
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			providerConfig:  configBodyForTest(t, `foobar = "foobar"`),
			wantErrorString: "schema contains a provider block",
		},
		"returns an error if the schema includes a provider attribute": {
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"provider": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			config: configBodyForTest(t, `
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			providerConfig:  configBodyForTest(t, `foobar = "foobar"`),
			wantErrorString: "schema contains a provider attribute",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			s := StateStore{
				Config: tc.config,
				Provider: &Provider{
					Config: tc.providerConfig,
				},
			}

			ssHash, pHash, diags := s.Hash(tc.schema, providerSchema)
			if diags.HasErrors() {
				if tc.wantErrorString == "" {
					t.Fatalf("unexpected error: %s", diags.Err())
				}
				if !strings.Contains(diags.Err().Error(), tc.wantErrorString) {
					t.Fatalf("expected %q to be in the returned error string but it's missing: %q", tc.wantErrorString, diags.Err())
				}

				return // early return if testing an error case
			}

			if !diags.HasErrors() && tc.wantErrorString != "" {
				t.Fatal("expected an error when generating a hash, but got none")
			}

			if ssHash == pHash {
				// These should not be equal, unless an error occurred and zero values were returned
				t.Fatalf("expected unique hashes for state_store and provider config, but they both have value: %d", ssHash)
			}
		})
	}
}

func configBodyForTest(t *testing.T, config string) hcl.Body {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(config), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("failure creating hcl.Body during test setup")
	}
	return f.Body
}
