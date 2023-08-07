// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package genconfig

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

func TestConfigGeneration(t *testing.T) {
	tcs := map[string]struct {
		schema   *configschema.Block
		addr     addrs.AbsResourceInstance
		provider addrs.LocalProviderConfig
		value    cty.Value
		expected string
	}{
		"simple_resource": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "tfcoremock",
			},
			value: cty.NilVal,
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  value = null          # OPTIONAL string
  list_block {          # OPTIONAL block
    nested_value = null # OPTIONAL string
  }
}`,
		},
		"simple_resource_with_state": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "tfcoremock",
			},
			value: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("D2320658"),
				"value": cty.StringVal("Hello, world!"),
				"list_block": cty.ObjectVal(map[string]cty.Value{
					"nested_value": cty.StringVal("Hello, solar system!"),
				}),
			}),
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  value = "Hello, world!"
  list_block {
    nested_value = "Hello, solar system!"
  }
}`,
		},
		"simple_resource_with_partial_state": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "tfcoremock",
			},
			value: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("D2320658"),
				"list_block": cty.ObjectVal(map[string]cty.Value{
					"nested_value": cty.StringVal("Hello, solar system!"),
				}),
			}),
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  value = null
  list_block {
    nested_value = "Hello, solar system!"
  }
}`,
		},
		"simple_resource_with_alternate_provider": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "mock",
			},
			value: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("D2320658"),
				"value": cty.StringVal("Hello, world!"),
				"list_block": cty.ObjectVal(map[string]cty.Value{
					"nested_value": cty.StringVal("Hello, solar system!"),
				}),
			}),
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  provider = mock
  value    = "Hello, world!"
  list_block {
    nested_value = "Hello, solar system!"
  }
}`,
		},
		"simple_resource_with_aliased_provider": {
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "tfcoremock",
				Alias:     "alternate",
			},
			value: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("D2320658"),
				"value": cty.StringVal("Hello, world!"),
				"list_block": cty.ObjectVal(map[string]cty.Value{
					"nested_value": cty.StringVal("Hello, solar system!"),
				}),
			}),
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  provider = tfcoremock.alternate
  value    = "Hello, world!"
  list_block {
    nested_value = "Hello, solar system!"
  }
}`,
		},
		"resource_with_nulls": {
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"single": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{},
							Nesting:    configschema.NestingSingle,
						},
						Required: true,
					},
					"list": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
							Nesting: configschema.NestingList,
						},
						Required: true,
					},
					"map": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
							Nesting: configschema.NestingMap,
						},
						Required: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_single": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
					// No configschema.NestingGroup example for this test, because this block type can never be null in state.
					"nested_list": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
					"nested_set": {
						Nesting: configschema.NestingSet,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
					"nested_map": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"nested_id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
			addr: addrs.AbsResourceInstance{
				Module: nil,
				Resource: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "tfcoremock_simple_resource",
						Name: "empty",
					},
					Key: nil,
				},
			},
			provider: addrs.LocalProviderConfig{
				LocalName: "tfcoremock",
			},
			value: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.StringVal("D2320658"),
				"single": cty.NullVal(cty.Object(map[string]cty.Type{})),
				"list": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				}))),
				"map": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				}))),
				"nested_single": cty.NullVal(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				})),
				"nested_list": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				})),
				"nested_set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				})),
				"nested_map": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"nested_id": cty.String,
				})),
			}),
			expected: `
resource "tfcoremock_simple_resource" "empty" {
  list   = null
  map    = null
  single = null
}`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.schema.InternalValidate()
			if err != nil {
				t.Fatalf("schema failed InternalValidate: %s", err)
			}
			contents, diags := GenerateResourceContents(tc.addr, tc.schema, tc.provider, tc.value)
			if len(diags) > 0 {
				t.Errorf("expected no diagnostics but found %s", diags)
			}

			got := WrapResourceContents(tc.addr, contents)
			want := strings.TrimSpace(tc.expected)
			if diff := cmp.Diff(got, want); len(diff) > 0 {
				t.Errorf("got:\n%s\nwant:\n%s\ndiff:\n%s", got, want, diff)
			}
		})
	}
}
