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
		config             hcl.Body
		providerConfig     hcl.Body
		schema             *configschema.Block
		wantErrorString    string
		wantProviderHash   int
		wantStateStoreHash int
	}{
		"ignores the provider block in config data, as long as the schema doesn't include it": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			providerConfig:     configBodyForTest(t, `foobar = "foobar"`),
			wantProviderHash:   2672365208,
			wantStateStoreHash: 3037430836,
		},
		"tolerates empty config block for the provider even when schema has Required field(s)": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
						# required field "foobar" is missing
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			providerConfig:     hcl.EmptyBody(),
			wantProviderHash:   2911589008,
			wantStateStoreHash: 3037430836,
		},
		"tolerates missing Required field(s) in state_store config": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
					  foobar = "foobar"
					}

					# required field "path" is missing
					workspace_dir = "foobar"
			}`),
			providerConfig:     configBodyForTest(t, `foobar = "foobar"`),
			wantProviderHash:   2672365208,
			wantStateStoreHash: 3453024478,
		},
		"returns errors when the config contains non-provider things that aren't in the schema": {
			schema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
					  foobar = "foobar"
					}
					unexpected_block {
					  foobar = "foobar"
					}
					unexpected_attr = "foobar"
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
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
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			providerConfig:  configBodyForTest(t, `foobar = "foobar"`),
			wantErrorString: `Protected block name "provider" in state store schema`,
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
			config: configBodyForTest(t, `state_store "foo" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			providerConfig:  configBodyForTest(t, `foobar = "foobar"`),
			wantErrorString: `Protected argument name "provider" in state store schema`,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			content, _, cfgDiags := tc.config.PartialContent(terraformBlockSchema)
			if len(cfgDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", cfgDiags)
			}
			var ssDiags hcl.Diagnostics
			s, ssDiags := decodeStateStoreBlock(content.Blocks.OfType("state_store")[0])
			if len(ssDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", ssDiags)
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

			if ssHash != tc.wantStateStoreHash {
				t.Fatalf("expected hash for state_store to be %d, but got %d", tc.wantStateStoreHash, ssHash)
			}
			if pHash != tc.wantProviderHash {
				t.Fatalf("expected hash for provider to be %d, but got %d", tc.wantProviderHash, pHash)
			}
		})
	}
}

func configBodyForTest(t *testing.T, config string) hcl.Body {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(config), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("failure creating hcl.Body during test setup: %s", diags.Error())
	}
	return f.Body
}
