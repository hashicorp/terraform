// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
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

	// schema from GetProviderSchema for the given state store
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
			config: SynthBody("TestStateStore_Hash", map[string]cty.Value{
				"provider": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),

				"path":          cty.StringVal("mystate.tfstate"),
				"workspace_dir": cty.StringVal("foobar"),
			}),
		},
		"Hash method returns errors when the config contains non-provider things that aren't in the schema": {
			schema: goodSchema,
			config: SynthBody("TestStateStore_Hash", map[string]cty.Value{
				"unexpected_block": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"unexpected_attr": cty.StringVal("mystate.tfstate"),

				"path":          cty.StringVal("mystate.tfstate"),
				"workspace_dir": cty.StringVal("foobar"),
			}),
			wantErrorString: "config contained unexpected values: unexpected_attr, unexpected_block",
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
			config: SynthBody("TestStateStore_Hash", map[string]cty.Value{
				// no provider block here
				"path":          cty.StringVal("mystate.tfstate"),
				"workspace_dir": cty.StringVal("foobar"),
			}),
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
			config: SynthBody("TestStateStore_Hash", map[string]cty.Value{
				// no provider block here
				"path":          cty.StringVal("mystate.tfstate"),
				"workspace_dir": cty.StringVal("foobar"),
			}),
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
