// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestNewQueryStart(t *testing.T) {
	makeAddr := func(resType, resName string) addrs.AbsResourceInstance {
		t.Helper()

		return addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: resType,
				Name: resName,
			},
			Key: addrs.NoKey,
		}.Absolute(addrs.RootModuleInstance)
	}

	tests := []struct {
		name        string
		addr        addrs.AbsResourceInstance
		inputConfig cty.Value
		schema      *configschema.Block
		want        QueryStart
	}{
		{
			name: "No sensitivity",
			addr: makeAddr("test_resource", "foo"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"foo":   cty.StringVal("bar"),
				"count": cty.NumberIntVal(1),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo":   {Type: cty.String, Optional: true},
					"count": {Type: cty.Number, Optional: true},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.foo",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{},
			},
		},
		{
			name: "Sensitivity via Value Marks (top level)",
			addr: makeAddr("test_resource", "secret_val"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"api_key": cty.StringVal("12345").Mark(marks.Sensitive),
				"public":  cty.StringVal("visible"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"api_key": {Type: cty.String, Optional: true},
					"public":  {Type: cty.String, Optional: true},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.secret_val",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{".api_key"},
			},
		},
		{
			name: "Sensitivity via Schema Definition",
			addr: makeAddr("test_resource", "schema_secret"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"password": cty.StringVal("hunter2"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"password": {Type: cty.String, Optional: true, Sensitive: true},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.schema_secret",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{".password"},
			},
		},
		{
			name: "Nested Map Value Sensitivity",
			addr: makeAddr("test_resource", "nested_map"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"env":    cty.StringVal("prod"),
					"secret": cty.StringVal("hidden").Mark(marks.Sensitive),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"tags": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.nested_map",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{`.tags["secret"]`},
			},
		},
		{
			name: "Nested List Value Sensitivity",
			addr: makeAddr("test_resource", "nested_list"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("one"),
					cty.StringVal("two").Mark(marks.Sensitive),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list": {Type: cty.List(cty.String), Optional: true},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.nested_list",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{".list[1]"},
			},
		},
		{
			name: "Complex Nested Schema Sensitivity (Nested Block)",
			addr: makeAddr("test_resource", "complex_schema"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"config_block": cty.ObjectVal(map[string]cty.Value{
					"token": cty.StringVal("abc"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"config_block": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"token": {Type: cty.String, Sensitive: true, Optional: true},
							},
						},
					},
				},
			},
			want: QueryStart{
				Address:                 "test_resource.complex_schema",
				ResourceType:            "test_resource",
				SensitiveAttributePaths: []string{".config_block.token"},
			},
		},
		{
			name: "Mixed: Schema Sensitive AND Value Marked",
			addr: makeAddr("test_resource", "mixed"),
			inputConfig: cty.ObjectVal(map[string]cty.Value{
				"double_secret": cty.StringVal("x").Mark(marks.Sensitive),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"double_secret": {Type: cty.String, Sensitive: true, Optional: true},
				},
			},
			want: QueryStart{
				Address:      "test_resource.mixed",
				ResourceType: "test_resource",
				// We expect NO duplicates if both sources flag it.
				SensitiveAttributePaths: []string{".double_secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewQueryStart(tt.addr, tt.inputConfig, tt.schema)

			if got.Address != tt.want.Address {
				t.Errorf("Address = %q, want %q", got.Address, tt.want.Address)
			}
			if got.ResourceType != tt.want.ResourceType {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tt.want.ResourceType)
			}

			// Sort slices for deterministic comparison
			sort.Strings(got.SensitiveAttributePaths)
			sort.Strings(tt.want.SensitiveAttributePaths)

			if !reflect.DeepEqual(got.SensitiveAttributePaths, tt.want.SensitiveAttributePaths) {
				t.Errorf("SensitiveInputConfig mismatch:\nGot:  %v\nWant: %v", got.SensitiveAttributePaths, tt.want.SensitiveAttributePaths)
			}
		})
	}
}
