// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// The Hash method assumes that the state_store schema doesn't include a provider block,
// and it requires calling code to remove the nested provider block from state_store config data.
func TestStateStore_Hash(t *testing.T) {

	// Normally these schemas would come from a provider's GetProviderSchema data
	exampleStateStoreSchema := &configschema.Block{
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
	exampleProviderSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foobar": {
				Type:     cty.String,
				Required: true,
			},
		},
	}

	// These values are all coupled.
	// The test case below asserts that given these inputs, the expected hash is returned.
	exampleProviderVersion := version.Must(version.NewSemver("1.2.3"))
	exampleProviderAddr := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "foobar")
	exampleConfig := configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`)
	exampleHash := 614398732
	t.Run("example happy path with all attrs set in the configuration", func(t *testing.T) {
		// Construct a configs.StateStore for the test.
		content, _, cfgDiags := exampleConfig.PartialContent(terraformBlockSchema)
		if len(cfgDiags) > 0 {
			t.Fatalf("unexpected diagnostics: %s", cfgDiags)
		}
		var ssDiags hcl.Diagnostics
		s, ssDiags := decodeStateStoreBlock(content.Blocks.OfType("state_store")[0])
		if len(ssDiags) > 0 {
			t.Fatalf("unexpected diagnostics: %s", ssDiags)
		}
		s.ProviderAddr = exampleProviderAddr

		// Test Hash method.
		gotHash, diags := s.Hash(exampleStateStoreSchema, exampleProviderSchema, exampleProviderVersion)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		if gotHash != exampleHash {
			t.Fatalf("expected hash for state_store to be %d, but got %d", exampleHash, gotHash)
		}
	})

	// Test cases each change a single input that affects the output hash
	// Assertions check that the output hash doesn't match the hash above, following the changed input.
	cases := map[string]struct {
		config           hcl.Body
		stateStoreSchema *configschema.Block
		providerVersion  *version.Version
		providerAddr     tfaddr.Provider
	}{
		"changing the state store type affects the hash value": {
			config: configBodyForTest(t, `state_store "foobar_CHANGED_VALUE_HERE" {
					provider "foobar" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
		},
		"changing the provider affects the hash value": {
			providerAddr: tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "different-provider"),
			config: configBodyForTest(t, `state_store "different-provider_fs" {
					provider "different-provider" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
		},
		"changing the provider version affects the hash value": {
			providerVersion: version.Must(version.NewSemver("9.9.9")),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// If a test case doesn't set an override for these inputs,
			// instead use a default value from the example above.
			var config hcl.Body
			var schema *configschema.Block
			var providerVersion *version.Version
			var providerAddr tfaddr.Provider
			if tc.config == nil {
				config = exampleConfig
			} else {
				config = tc.config
			}
			if tc.stateStoreSchema == nil {
				schema = exampleStateStoreSchema
			} else {
				schema = tc.stateStoreSchema
			}
			if tc.providerVersion == nil {
				providerVersion = exampleProviderVersion
			} else {
				providerVersion = tc.providerVersion
			}
			if tc.providerAddr.IsZero() {
				providerAddr = exampleProviderAddr
			} else {
				providerAddr = tc.providerAddr
			}

			// Construct a configs.StateStore for the test.
			content, _, cfgDiags := config.PartialContent(terraformBlockSchema)
			if len(cfgDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", cfgDiags)
			}
			var ssDiags hcl.Diagnostics
			s, ssDiags := decodeStateStoreBlock(content.Blocks.OfType("state_store")[0])
			if len(ssDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", ssDiags)
			}
			s.ProviderAddr = providerAddr

			// Test Hash method.
			gotHash, diags := s.Hash(schema, exampleProviderSchema, providerVersion)
			if diags.HasErrors() {
				t.Fatalf("unexpected error: %s", diags.Err())
			}
			if gotHash == exampleHash {
				t.Fatal("expected hash for state_store to be different from the example due to a changed input, but it matched.")
			}
		})
	}
}

func TestStateStore_Hash_edgeCases(t *testing.T) {
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
	providerAddr := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "foobar")
	providerVersion := version.Must(version.NewSemver("1.2.3"))
	config := configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
						foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`)

	cases := map[string]struct {
		config          hcl.Body
		providerAddr    tfaddr.Provider
		providerVersion *version.Version
		reattachConfig  string
	}{
		"tolerates empty config block for the provider even when schema has Required field(s)": {
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
						# required field "foobar" is missing
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			providerAddr:    providerAddr,
			providerVersion: providerVersion,
		},
		"tolerates missing Required field(s) in state_store config": {
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					}

					# required field "path" is missing
					workspace_dir = "foobar"
			}`),
			providerAddr:    providerAddr,
			providerVersion: providerVersion,
		},
		"tolerates missing provider version data when using a builtin provider": {
			config:          config,
			providerAddr:    tfaddr.NewProvider(tfaddr.BuiltInProviderHost, "hashicorp", "foobar"), // Builtin
			providerVersion: nil,                                                                   // No version
		},
		"tolerates missing provider version data when using a reattached provider": {
			config:          config,
			providerAddr:    providerAddr,
			providerVersion: nil, // No version
			reattachConfig: `{
				"foobar": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			if tc.reattachConfig != "" {
				t.Setenv("TF_REATTACH_PROVIDERS", tc.reattachConfig)
			}

			// Construct a configs.StateStore for the test.
			content, _, cfgDiags := config.PartialContent(terraformBlockSchema)
			if len(cfgDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", cfgDiags)
			}
			var ssDiags hcl.Diagnostics
			s, ssDiags := decodeStateStoreBlock(content.Blocks.OfType("state_store")[0])
			if len(ssDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", ssDiags)
			}
			s.ProviderAddr = tc.providerAddr

			// Test Hash method.
			_, diags := s.Hash(stateStoreSchema, providerSchema, tc.providerVersion)
			if diags.HasErrors() {
				t.Fatalf("unexpected error: %s", diags.Err())
			}
		})
	}
}

func TestStateStore_Hash_errorConditions(t *testing.T) {
	// Normally these schemas would come from a provider's GetProviderSchema data
	exampleStateStoreSchema := &configschema.Block{
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
	exampleProviderSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foobar": {
				Type:     cty.String,
				Required: true,
			},
		},
	}
	exampleProviderVersion := version.Must(version.NewSemver("1.2.3"))

	// Cases where an error would occur
	cases := map[string]struct {
		config           hcl.Body
		stateStoreSchema *configschema.Block
		providerVersion  *version.Version
		wantErrorString  string
	}{
		"returns errors when the state_store config doesn't match the schema": {
			providerVersion:  exampleProviderVersion,
			stateStoreSchema: exampleStateStoreSchema,
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
			providerVersion:  exampleProviderVersion,
			stateStoreSchema: exampleStateStoreSchema,
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
			providerVersion: exampleProviderVersion,
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
			providerVersion: exampleProviderVersion,
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
		"returns an error if the provider version is missing when using a non-builtin, non-reattached provider": {
			providerVersion:  nil, // No value provided in this test case
			stateStoreSchema: exampleStateStoreSchema,
			config: configBodyForTest(t, `state_store "foobar_fs" {
					provider "foobar" {
					  foobar = "foobar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"
			}`),
			wantErrorString: `Provider version data was missing during hash generation`,
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
			s.ProviderAddr = tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "foobar")

			// Test Hash method.
			_, diags := s.Hash(tc.stateStoreSchema, exampleProviderSchema, tc.providerVersion)
			if !diags.HasErrors() {
				t.Fatal("expected error but got none")
			}
			if !strings.Contains(diags.Err().Error(), tc.wantErrorString) {
				t.Fatalf("expected error to contain %q but got: %s", tc.wantErrorString, diags.Err())
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
