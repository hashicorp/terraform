package terraform

import (
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/zclconf/go-cty/cty"
)

func TestResource(t *testing.T) {
	if err := dataSourceRemoteStateGetSchema().Block.InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestState_basic(t *testing.T) {
	var tests = map[string]struct {
		Config cty.Value
		Want   cty.Value
		Err    bool
	}{
		"basic": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/basic.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/basic.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"workspace": cty.NullVal(cty.String),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"complex outputs": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/complex_outputs.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/complex_outputs.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"computed_map": cty.MapVal(map[string]cty.Value{
						"key1": cty.StringVal("value1"),
					}),
					"computed_set": cty.ListVal([]cty.Value{
						cty.StringVal("setval1"),
						cty.StringVal("setval2"),
					}),
					"map": cty.MapVal(map[string]cty.Value{
						"key":  cty.StringVal("test"),
						"test": cty.StringVal("test"),
					}),
					"set": cty.ListVal([]cty.Value{
						cty.StringVal("test1"),
						cty.StringVal("test2"),
					}),
				}),
				"workspace": cty.NullVal(cty.String),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"null outputs": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/null_outputs.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/null_outputs.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"map":  cty.NullVal(cty.DynamicPseudoType),
					"list": cty.NullVal(cty.DynamicPseudoType),
				}),
				"workspace": cty.NullVal(cty.String),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"defaults": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/empty.tfstate"),
				}),
				"defaults": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/empty.tfstate"),
				}),
				"defaults": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"workspace": cty.NullVal(cty.String),
			}),
			false,
		},
		"missing": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/missing.tfstate"), // intentionally not present on disk
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./test-fixtures/missing.tfstate"),
				}),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"outputs":   cty.EmptyObjectVal,
				"workspace": cty.NullVal(cty.String),
			}),
			true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			schema := dataSourceRemoteStateGetSchema().Block
			config, err := schema.CoerceValue(test.Config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got, diags := dataSourceRemoteStateRead(&config)

			if test.Err {
				if !diags.HasErrors() {
					t.Fatal("succeeded; want error")
				}
			} else if diags.HasErrors() {
				t.Fatalf("unexpected errors: %s", diags.Err())
			}

			if !test.Want.RawEquals(got) {
				t.Errorf("wrong result\nconfig: %sgot: %swant: %s", dump.Value(config), dump.Value(got), dump.Value(test.Want))
			}
		})
	}
}
