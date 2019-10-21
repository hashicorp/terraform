package terraform

import (
	"github.com/hashicorp/terraform/tfdiags"
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/hashicorp/terraform/backend"
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
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"workspace": cty.StringVal(backend.DefaultStateName),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"complex outputs": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/complex_outputs.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/complex_outputs.tfstate"),
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
				"workspace": cty.StringVal(backend.DefaultStateName),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"null outputs": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/null_outputs.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/null_outputs.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"map":  cty.NullVal(cty.DynamicPseudoType),
					"list": cty.NullVal(cty.DynamicPseudoType),
				}),
				"workspace": cty.StringVal(backend.DefaultStateName),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"defaults": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/empty.tfstate"),
				}),
				"defaults": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/empty.tfstate"),
				}),
				"defaults": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"workspace": cty.StringVal(backend.DefaultStateName),
			}),
			false,
		},
		"missing": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/missing.tfstate"), // intentionally not present on disk
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/missing.tfstate"),
				}),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"outputs":   cty.EmptyObjectVal,
				"workspace": cty.StringVal(backend.DefaultStateName),
			}),
			true,
		},
		"wrong type for config": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config":  cty.StringVal("nope"),
			}),
			cty.NilVal,
			true,
		},
		"wrong type for config with unknown backend": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.UnknownVal(cty.String),
				"config":  cty.StringVal("nope"),
			}),
			cty.NilVal,
			true,
		},
		"wrong type for config with unknown config": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config":  cty.UnknownVal(cty.String),
			}),
			cty.NilVal,
			true,
		},
		"wrong type for defaults": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"defaults": cty.StringVal("nope"),
			}),
			cty.NilVal,
			true,
		},
		"config as map": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.MapVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/empty.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.MapVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/empty.tfstate"),
				}),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"outputs":   cty.EmptyObjectVal,
				"workspace": cty.StringVal(backend.DefaultStateName),
			}),
			false,
		},
		"defaults as map": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"defaults": cty.MapValEmpty(cty.String),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"defaults": cty.MapValEmpty(cty.String),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"workspace": cty.StringVal(backend.DefaultStateName),
			}),
			false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			schema := dataSourceRemoteStateGetSchema().Block
			config, err := schema.CoerceValue(test.Config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			diags := dataSourceRemoteStateValidate(config)

			var got cty.Value
			if !diags.HasErrors() && config.IsWhollyKnown() {
				var moreDiags tfdiags.Diagnostics
				got, moreDiags = dataSourceRemoteStateRead(config)
				diags = diags.Append(moreDiags)
			}

			if test.Err {
				if !diags.HasErrors() {
					t.Fatal("succeeded; want error")
				}
			} else if diags.HasErrors() {
				t.Fatalf("unexpected errors: %s", diags.Err())
			}

			if test.Want != cty.NilVal && !test.Want.RawEquals(got) {
				t.Errorf("wrong result\nconfig: %sgot:    %swant:   %s", dump.Value(config), dump.Value(got), dump.Value(test.Want))
			}
		})
	}
}
