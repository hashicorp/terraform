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
	//       foo = "bar"
	//     }

	//     # Attributes for configuring the state store
	//     path          = "mystate.tfstate"
	//     workspace_dir = "foobar"
	//   }
	// }

	// Normally this schema would come from a provider's GetProviderSchema data
	goodSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"path": {
				Type:     cty.String,
				Optional: true,
			},
			"workspace_dir": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	cases := map[string]struct {
		config          hcl.Body
		schema          *configschema.Block
		wantErrorString string
	}{
		"Hash method ignores the provider block in config data": {
			schema: goodSchema,
			config: configBodyForTest(t, `
					provider "foobar" {
					  foo = "bar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
		},
		"Hash method returns errors when the config contains non-provider things that aren't in the schema": {
			schema: goodSchema,
			config: configBodyForTest(t, `
					unexpected_block {
					  foo = "bar"
					}
					unexpected_attr = "foobar"
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			wantErrorString: "Unsupported argument",
		},
		"Hash method returns an error if the schema includes a provider block": {
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
					  foo = "bar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			wantErrorString: "schema contains a provider block",
		},
		"Hash method returns an error if the schema includes a provider attribute": {
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
					  foo = "bar"
					}
					path          = "mystate.tfstate"
					workspace_dir = "foobar"`),
			wantErrorString: "schema contains a provider attribute",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			s := StateStore{
				Config: tc.config,
			}

			_, diags := s.Hash(tc.schema)
			if !diags.HasErrors() && tc.wantErrorString != "" {
				t.Fatal("expected an error when generating a hash, but got none")
			}
			if diags.HasErrors() && tc.wantErrorString == "" {
				t.Fatalf("unexpected error: %s", diags.Err())
			}
			if diags.HasErrors() && !strings.Contains(diags.Err().Error(), tc.wantErrorString) {
				t.Fatalf("expected %q to be in the returned error string but it's missing: %q", tc.wantErrorString, diags.Err())
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
