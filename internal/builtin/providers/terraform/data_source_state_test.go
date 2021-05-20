package terraform

import (
	"fmt"
	"log"
	"testing"

	"github.com/apparentlymart/go-dump/dump"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"workspace": cty.NullVal(cty.String),
			}),
			false,
		},
		"workspace": {
			cty.ObjectVal(map[string]cty.Value{
				"backend":   cty.StringVal("local"),
				"workspace": cty.StringVal(backend.DefaultStateName),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend":   cty.StringVal("local"),
				"workspace": cty.StringVal(backend.DefaultStateName),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"defaults": cty.NullVal(cty.DynamicPseudoType),
			}),
			false,
		},
		"_local": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("_local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("_local"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
				"outputs": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"workspace": cty.NullVal(cty.String),
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
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"workspace": cty.NullVal(cty.String),
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
					"map":  cty.NullVal(cty.Map(cty.String)),
					"list": cty.NullVal(cty.List(cty.String)),
				}),
				"defaults":  cty.NullVal(cty.DynamicPseudoType),
				"workspace": cty.NullVal(cty.String),
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
				"workspace": cty.NullVal(cty.String),
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
				"workspace": cty.NullVal(cty.String),
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
				"workspace": cty.NullVal(cty.String),
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
				"workspace": cty.NullVal(cty.String),
			}),
			false,
		},
		"nonexistent backend": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("nonexistent"),
				"config": cty.ObjectVal(map[string]cty.Value{
					"path": cty.StringVal("./testdata/basic.tfstate"),
				}),
			}),
			cty.NilVal,
			true,
		},
		"null config": {
			cty.ObjectVal(map[string]cty.Value{
				"backend": cty.StringVal("local"),
				"config":  cty.NullVal(cty.DynamicPseudoType),
			}),
			cty.NilVal,
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

func TestState_validation(t *testing.T) {
	// The main test TestState_basic covers both validation and reading of
	// state snapshots, so this additional test is here only to verify that
	// the validation step in isolation does not attempt to configure
	// the backend.
	overrideBackendFactories = map[string]backend.InitFn{
		"failsconfigure": func() backend.Backend {
			return backendFailsConfigure{}
		},
	}
	defer func() {
		// undo our overrides so we won't affect other tests
		overrideBackendFactories = nil
	}()

	schema := dataSourceRemoteStateGetSchema().Block
	config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"backend": cty.StringVal("failsconfigure"),
		"config":  cty.EmptyObjectVal,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	diags := dataSourceRemoteStateValidate(config)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}
}

type backendFailsConfigure struct{}

func (b backendFailsConfigure) ConfigSchema() *configschema.Block {
	log.Printf("[TRACE] backendFailsConfigure.ConfigSchema")
	return &configschema.Block{} // intentionally empty configuration schema
}

func (b backendFailsConfigure) PrepareConfig(given cty.Value) (cty.Value, tfdiags.Diagnostics) {
	// No special actions to take here
	return given, nil
}

func (b backendFailsConfigure) Configure(config cty.Value) tfdiags.Diagnostics {
	log.Printf("[TRACE] backendFailsConfigure.Configure(%#v)", config)
	var diags tfdiags.Diagnostics
	diags = diags.Append(fmt.Errorf("Configure should never be called"))
	return diags
}

func (b backendFailsConfigure) StateMgr(workspace string) (statemgr.Full, error) {
	return nil, fmt.Errorf("StateMgr not implemented")
}

func (b backendFailsConfigure) DeleteWorkspace(name string) error {
	return fmt.Errorf("DeleteWorkspace not implemented")
}

func (b backendFailsConfigure) Workspaces() ([]string, error) {
	return nil, fmt.Errorf("Workspaces not implemented")
}
