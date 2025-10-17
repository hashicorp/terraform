// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfaddr "github.com/hashicorp/terraform-registry-address"
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

	// These two values are coupled.
	exampleConfig := configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`)
	exampleHash := 33464751

	cases := map[string]struct {
		config           hcl.Body
		stateStoreSchema *configschema.Block
		providerAddr     tfaddr.Provider
		wantErrorString  string
		wantHash         int
	}{
		"example happy path with all attrs set in the configuration": {
			stateStoreSchema: stateStoreSchema,
			config:           exampleConfig,
			wantHash:         exampleHash,
		},
		"changing the state store type affects the hash value": {
			stateStoreSchema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_CHANGED_VALUE_HERE" {
					provider "foobar" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantHash: 559959421, // Differs from `exampleHash`
		},
		"changing the provider affects the hash value": {
			stateStoreSchema: stateStoreSchema,
			providerAddr:     tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "different-provider"),
			config: configBodyForTest(t, `state_store "different-provider_fs" {
					provider "different-provider" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantHash: 1672894798, // Differs from `exampleHash`
		},
		"tolerates empty config block for the provider even when schema has Required field(s)": {
			stateStoreSchema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
						# required field "foobar" is missing
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantHash: 3558227459,
		},
		"tolerates missing Required field(s) in state_store config": {
			stateStoreSchema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					}

					# required field "path" is missing
					workspace_dir = "foobar"
			}`),
			wantHash: 3682853451,
		},
		"returns errors when the state_store config doesn't match the schema": {
			stateStoreSchema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_fs" {
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
			wantErrorString: "Unsupported argument",
		},
		"returns errors when the provider config doesn't match the schema": {
			stateStoreSchema: stateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					  unexpected_attr = "foobar"
					  unexpected_block {
					    foobar = "foobar"
					  }
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantErrorString: "Unsupported argument",
		},
		"returns an error if the state_store schema includes a provider block": {
			stateStoreSchema: &configschema.Block{
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
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantErrorString: `Protected block name "provider" in state store schema`,
		},
		"returns an error if the state_store schema includes a provider attribute": {
			stateStoreSchema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"provider": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantErrorString: `Protected argument name "provider" in state store schema`,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Construct a configs.StateStore for the test.
			content, _, cfgDiags := tc.config.PartialContent(terraformBlockSchema)
			if len(cfgDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", cfgDiags)
			}
			var ssDiags hcl.Diagnostics
			s, ssDiags := decodeStateStoreBlock(content.Blocks.OfType("state_store")[0])
			if len(ssDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", ssDiags)
			}
			// Add provider addr
			if tc.providerAddr.IsZero() {
				s.ProviderAddr = tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "foobar")
			} else {
				s.ProviderAddr = tc.providerAddr
			}

			// Test Hash method.
			gotHash, diags := s.Hash(tc.stateStoreSchema, providerSchema)
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

			if gotHash != tc.wantHash {
				t.Fatalf("expected hash for state_store to be %d, but got %d", tc.wantHash, gotHash)
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
