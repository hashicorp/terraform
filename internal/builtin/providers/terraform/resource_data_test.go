// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestManagedDataValidate(t *testing.T) {
	cfg := map[string]cty.Value{
		"input":            cty.NullVal(cty.DynamicPseudoType),
		"output":           cty.NullVal(cty.DynamicPseudoType),
		"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
		"id":               cty.NullVal(cty.String),
	}

	// empty
	req := providers.ValidateResourceConfigRequest{
		TypeName: "terraform_data",
		Config:   cty.ObjectVal(cfg),
	}

	resp := validateDataStoreResourceConfig(req)
	if resp.Diagnostics.HasErrors() {
		t.Error("empty config error:", resp.Diagnostics.ErrWithWarnings())
	}

	// invalid computed values
	cfg["output"] = cty.StringVal("oops")
	req.Config = cty.ObjectVal(cfg)

	resp = validateDataStoreResourceConfig(req)
	if !resp.Diagnostics.HasErrors() {
		t.Error("expected error")
	}

	msg := resp.Diagnostics.Err().Error()
	if !strings.Contains(msg, "attribute is read-only") {
		t.Error("unexpected error", msg)
	}
}

func TestManagedDataUpgradeState(t *testing.T) {
	rawState := `{
	"id": "not-quite-unique",
	"input": {
		"value": "input",
		"type": "string"
	},
	"output": {
		"value": "input",
		"type": "string"
	},
	"triggers_replace": {
		"value": [
			"a",
			"b"
		],
		"type": [
			"list",
			"string"
		]
	}
}`

	upgradedState, err := dataStoreResourceSchema().Body.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"input":  cty.StringVal("input"),
		"output": cty.StringVal("input"),
		"triggers_replace": cty.ListVal([]cty.Value{
			cty.StringVal("a"), cty.StringVal("b"),
		}),
		"id": cty.StringVal("not-quite-unique"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	req := providers.UpgradeResourceStateRequest{
		TypeName:     "terraform_data",
		RawStateJSON: []byte(rawState),
	}

	resp := upgradeDataStoreResourceState(req)
	if resp.Diagnostics.HasErrors() {
		t.Error("upgrade state error:", resp.Diagnostics.ErrWithWarnings())
	}

	if !resp.UpgradedState.RawEquals(upgradedState) {
		t.Errorf("prior state was:\n%s\nupgraded state is:\n%#v\n", rawState, resp.UpgradedState)
	}
}

func TestManagedDataRead(t *testing.T) {
	req := providers.ReadResourceRequest{
		TypeName: "terraform_data",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"input":  cty.StringVal("input"),
			"output": cty.StringVal("input"),
			"triggers_replace": cty.ListVal([]cty.Value{
				cty.StringVal("a"), cty.StringVal("b"),
			}),
			"id": cty.StringVal("not-quite-unique"),
		}),
	}

	resp := readDataStoreResourceState(req)
	if resp.Diagnostics.HasErrors() {
		t.Fatal("unexpected error", resp.Diagnostics.ErrWithWarnings())
	}

	if !resp.NewState.RawEquals(req.PriorState) {
		t.Errorf("prior state was:\n%#v\nnew state is:\n%#v\n", req.PriorState, resp.NewState)
	}
}

func TestManagedDataPlan(t *testing.T) {
	schema := dataStoreResourceSchema().Body
	ty := schema.ImpliedType()

	for name, tc := range map[string]struct {
		prior    cty.Value
		proposed cty.Value
		planned  cty.Value
	}{
		"create": {
			prior: cty.NullVal(ty),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.DynamicPseudoType),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.NullVal(cty.String),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.DynamicPseudoType),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.UnknownVal(cty.String).RefineNotNull(),
			}),
		},

		"create-typed-null-input": {
			prior: cty.NullVal(ty),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.String),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.NullVal(cty.String),
				"sensitive": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.Number),
					"output": cty.NullVal(cty.DynamicPseudoType),
				}),
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.Number),
					"output": cty.NullVal(cty.DynamicPseudoType),
				}),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.String),
				"output":           cty.NullVal(cty.String),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.UnknownVal(cty.String).RefineNotNull(),
				"sensitive": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.Number),
					"output": cty.NullVal(cty.Number),
				}),
				"write_only": cty.ObjectVal(map[string]cty.Value{
					// write-only values are always returned as null
					"input":  cty.NullVal(cty.DynamicPseudoType),
					"output": cty.NullVal(cty.Number),
				}),
			}),
		},

		"create-output": {
			prior: cty.NullVal(ty),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.NullVal(cty.String),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.UnknownVal(cty.String),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.UnknownVal(cty.String).RefineNotNull(),
			}),
		},

		"update-input": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
				"sensitive": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.StringVal("input"),
					"output": cty.StringVal("input"),
				}),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.UnknownVal(cty.List(cty.String)),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
				"sensitive": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.UnknownVal(cty.List(cty.String)),
					"output": cty.StringVal("input"),
				}),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.UnknownVal(cty.List(cty.String)),
				"output":           cty.UnknownVal(cty.List(cty.String)),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
				"sensitive": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.UnknownVal(cty.List(cty.String)),
					"output": cty.UnknownVal(cty.List(cty.String)),
				}),
			}),
		},

		"update-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.StringVal("new-value"),
				"id":               cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.UnknownVal(cty.String),
				"triggers_replace": cty.StringVal("new-value"),
				"id":               cty.UnknownVal(cty.String).RefineNotNull(),
			}),
		},

		"update-input-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.StringVal("input"),
				"output": cty.StringVal("input"),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("value"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output": cty.StringVal("input"),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("new value"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output": cty.UnknownVal(cty.List(cty.String)),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("new value"),
				}),
				"id": cty.UnknownVal(cty.String).RefineNotNull(),
			}),
		},

		"update-wo-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(1),
					"output":  cty.StringVal("ephem"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.StringVal("ephem"),
					"version": cty.NumberIntVal(2),
					"output":  cty.StringVal("ephem"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(2),
					"output":  cty.UnknownVal(cty.String),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},

		"update-wo-auto": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.DynamicPseudoType),
					"output": cty.StringVal("ephem_2"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.StringVal("ephem_1"),
					"output": cty.StringVal("ephem_2"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.DynamicPseudoType),
					"output": cty.UnknownVal(cty.String),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},

		"no-update-wo-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(1),
					"output":  cty.StringVal("ephem"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.StringVal("ephem 2"),
					"version": cty.NumberIntVal(1),
					"output":  cty.StringVal("ephem"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(1),
					"output":  cty.StringVal("ephem"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},

		"no-update-wo-auto": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.DynamicPseudoType),
					"output": cty.StringVal("ephem_2"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			proposed: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.StringVal("ephem_2"),
					"output": cty.StringVal("ephem_2"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":  cty.NullVal(cty.DynamicPseudoType),
					"output": cty.StringVal("ephem_2"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},
	} {
		t.Run("plan-"+name, func(t *testing.T) {
			req := providers.PlanResourceChangeRequest{
				TypeName:         "terraform_data",
				PriorState:       mustCoerceManagedData(t, tc.prior),
				ProposedNewState: mustCoerceManagedData(t, tc.proposed),
			}

			resp := planDataStoreResourceChange(req)
			if resp.Diagnostics.HasErrors() {
				t.Fatal(resp.Diagnostics.ErrWithWarnings())
			}
			expectedPlanned := mustCoerceManagedData(t, tc.planned)

			if !resp.PlannedState.RawEquals(expectedPlanned) {
				t.Errorf("expected:\n%#v\ngot:\n%#v\n", expectedPlanned, resp.PlannedState)
			}
		})
	}
}

func mustCoerceManagedData(t *testing.T, v cty.Value) cty.Value {
	schema := dataStoreResourceSchema().Body
	v, err := schema.CoerceValue(v)
	if err != nil {
		t.Fatalf("failed to coerce value: %s", err)
	}
	return v
}

func TestManagedDataApply(t *testing.T) {
	testUUIDHook = func() string {
		return "not-quite-unique"
	}
	defer func() {
		testUUIDHook = nil
	}()

	schema := dataStoreResourceSchema().Body
	ty := schema.ImpliedType()

	for name, tc := range map[string]struct {
		prior   cty.Value
		planned cty.Value
		state   cty.Value
	}{
		"create": {
			prior: cty.NullVal(ty),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.DynamicPseudoType),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.UnknownVal(cty.String),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.NullVal(cty.DynamicPseudoType),
				"output":           cty.NullVal(cty.DynamicPseudoType),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
		},

		"create-output": {
			prior: cty.NullVal(ty),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.UnknownVal(cty.String),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.UnknownVal(cty.String),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
		},

		"update-input": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output":           cty.UnknownVal(cty.List(cty.String)),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output":           cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
		},

		"update-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
				"id":               cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.UnknownVal(cty.String),
				"triggers_replace": cty.StringVal("new-value"),
				"id":               cty.UnknownVal(cty.String),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"input":            cty.StringVal("input"),
				"output":           cty.StringVal("input"),
				"triggers_replace": cty.StringVal("new-value"),
				"id":               cty.StringVal("not-quite-unique"),
			}),
		},

		"update-input-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.StringVal("input"),
				"output": cty.StringVal("input"),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("value"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output": cty.UnknownVal(cty.List(cty.String)),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("new value"),
				}),
				"id": cty.UnknownVal(cty.String),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"input":  cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"output": cty.ListVal([]cty.Value{cty.StringVal("new-input")}),
				"triggers_replace": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("new value"),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},

		"update-wo-trigger": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(1),
					"output":  cty.StringVal("ephem"),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.StringVal("new_ephem"),
					"version": cty.NumberIntVal(2),
					"output":  cty.UnknownVal(cty.String),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"version": cty.NumberIntVal(2),
					"output":  cty.StringVal("new_ephem"),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},

		"update-wo-auto": {
			prior: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"output":  cty.StringVal("ephem"),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			planned: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.StringVal("new_ephem"),
					"output":  cty.UnknownVal(cty.String),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
			state: cty.ObjectVal(map[string]cty.Value{
				"write_only": cty.ObjectVal(map[string]cty.Value{
					"input":   cty.NullVal(cty.DynamicPseudoType),
					"output":  cty.StringVal("new_ephem"),
					"replace": cty.NullVal(cty.Bool),
				}),
				"id": cty.StringVal("not-quite-unique"),
			}),
		},
	} {
		t.Run("apply-"+name, func(t *testing.T) {
			req := providers.ApplyResourceChangeRequest{
				TypeName:     "terraform_data",
				Config:       mustCoerceManagedData(t, tc.planned),
				PriorState:   mustCoerceManagedData(t, tc.prior),
				PlannedState: mustCoerceManagedData(t, tc.planned),
			}

			resp := applyDataStoreResourceChange(req)
			if resp.Diagnostics.HasErrors() {
				t.Fatal(resp.Diagnostics.ErrWithWarnings())
			}

			expected := mustCoerceManagedData(t, tc.state)

			if !resp.NewState.RawEquals(expected) {
				t.Errorf("expected:\n%#v\ngot:\n%#v\n", expected, resp.NewState)
			}
		})
	}
}

func TestMoveDataStoreResourceState_Id(t *testing.T) {
	t.Parallel()

	nullResourceStateValue := cty.ObjectVal(map[string]cty.Value{
		"id":       cty.StringVal("test"),
		"triggers": cty.NullVal(cty.Map(cty.String)),
	})
	nullResourceStateJSON, err := ctyjson.Marshal(nullResourceStateValue, nullResourceStateValue.Type())

	if err != nil {
		t.Fatalf("failed to marshal null resource state: %s", err)
	}

	req := providers.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/null",
		SourceStateJSON:       nullResourceStateJSON,
		SourceTypeName:        "null_resource",
		TargetTypeName:        "terraform_data",
	}
	resp := moveDataStoreResourceState(req)

	if resp.Diagnostics.HasErrors() {
		t.Errorf("unexpected diagnostics: %s", resp.Diagnostics.Err())
	}

	expected, err := dataStoreResourceSchema().Body.CoerceValue(cty.EmptyObjectVal)
	if err != nil {
		t.Fatal(err)
	}
	expectedMap := expected.AsValueMap()

	expectedMap["id"] = cty.StringVal("test")
	expectedTargetState := cty.ObjectVal(expectedMap)

	if !resp.TargetState.RawEquals(expectedTargetState) {
		t.Errorf("expected state was:\n%#v\ngot state is:\n%#v\n", expectedTargetState, resp.TargetState)
	}
}

func TestMoveResourceState_SourceProviderAddress(t *testing.T) {
	t.Parallel()

	req := providers.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/examplecorp/null",
	}
	resp := moveDataStoreResourceState(req)

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected diagnostics")
	}
}

func TestMoveResourceState_SourceTypeName(t *testing.T) {
	t.Parallel()

	req := providers.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/null",
		SourceTypeName:        "null_data_source",
	}
	resp := moveDataStoreResourceState(req)

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected diagnostics")
	}
}

func TestMoveDataStoreResourceState_Triggers(t *testing.T) {
	t.Parallel()

	nullResourceStateValue := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
		"triggers": cty.MapVal(map[string]cty.Value{
			"testkey": cty.StringVal("testvalue"),
		}),
	})
	nullResourceStateJSON, err := ctyjson.Marshal(nullResourceStateValue, nullResourceStateValue.Type())

	if err != nil {
		t.Fatalf("failed to marshal null resource state: %s", err)
	}

	req := providers.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/null",
		SourceStateJSON:       nullResourceStateJSON,
		SourceTypeName:        "null_resource",
		TargetTypeName:        "terraform_data",
	}
	resp := moveDataStoreResourceState(req)

	if resp.Diagnostics.HasErrors() {
		t.Errorf("unexpected diagnostics: %s", resp.Diagnostics.Err())
	}

	expected, err := dataStoreResourceSchema().Body.CoerceValue(cty.EmptyObjectVal)
	if err != nil {
		t.Fatal(err)
	}
	expectedMap := expected.AsValueMap()

	expectedMap["id"] = cty.StringVal("test")
	expectedMap["triggers_replace"] = cty.ObjectVal(map[string]cty.Value{
		"testkey": cty.StringVal("testvalue"),
	})
	expectedTargetState := cty.ObjectVal(expectedMap)

	if !resp.TargetState.RawEquals(expectedTargetState) {
		t.Errorf("expected state was:\n%#v\ngot state is:\n%#v\n", expectedTargetState, resp.TargetState)
	}
}
