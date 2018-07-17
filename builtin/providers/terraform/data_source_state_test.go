package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestResource(t *testing.T) {
	if err := dataSourceRemoteStateGetSchema().Block.InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestState_basic(t *testing.T) {
	var tests = []struct {
		Config cty.Value
		Want   cty.Value
		Err    bool
	}{
		{ // basic test
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
			}),
			false,
		},
		{ // complex outputs
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
					"computed_map": cty.ObjectVal(map[string]cty.Value{
						"key1": cty.StringVal("value1"),
					}),
					"computed_set": cty.TupleVal([]cty.Value{
						cty.StringVal("setval1"),
						cty.StringVal("setval2"),
					}),
					"map": cty.ObjectVal(map[string]cty.Value{
						"key":  cty.StringVal("test"),
						"test": cty.StringVal("test"),
					}),
					"set": cty.TupleVal([]cty.Value{
						cty.StringVal("test1"),
						cty.StringVal("test2"),
					}),
				}),
			}),
			false,
		},
		{ // null outputs
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
			}),
			false,
		},
		{ // defaults
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
			}),
			false,
		},
	}
	for _, test := range tests {
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
			t.Fatalf("unexpected error: %s", err)
		}

		if !got.RawEquals(test.Want) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
		}
	}
}
