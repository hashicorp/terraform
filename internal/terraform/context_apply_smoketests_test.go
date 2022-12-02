package terraform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2ApplySmokeTests_noData(t *testing.T) {
	cfg := testModule(t, "apply-smoketests-no-data")
	smokeTestAddr := addrs.SmokeTest{Name: "try"}
	smokeTestAddrConfig := smokeTestAddr.InModule(addrs.RootModule)
	smokeTestAddrAbs := smokeTestAddr.Absolute(addrs.RootModuleInstance)

	makePlan := func(t *testing.T, core *Context, aVal, bVal cty.Value) *plans.Plan {
		plan, diags := core.Plan(
			cfg, states.NewState(),
			SimplePlanOpts(plans.NormalMode, InputValues{
				"a": &InputValue{
					Value:      aVal,
					SourceType: ValueFromCaller,
				},
				"b": &InputValue{
					Value:      bVal,
					SourceType: ValueFromCaller,
				},
			}),
		)
		assertNoDiagnostics(t, diags)

		// We should know know that the smoke test exists as a checkable
		// object, but we don't know its status yet because we'll decide
		// that only during apply.
		got := plan.Checks
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusUnknown,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusUnknown,
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after plan\n%s", diff)
		}

		return plan
	}

	t.Run("both_passing", func(t *testing.T) {
		core := testContext2(t, &ContextOpts{})
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("b"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusPass,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusPass,
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("pre failing", func(t *testing.T) {
		core := testContext2(t, &ContextOpts{})
		plan := makePlan(t, core, cty.StringVal("not a"), cty.StringVal("b"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"A isn't."},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("post failing", func(t *testing.T) {
		core := testContext2(t, &ContextOpts{})
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("not b"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"B isn't."},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("both failing", func(t *testing.T) {
		core := testContext2(t, &ContextOpts{})
		plan := makePlan(t, core, cty.StringVal("not a"), cty.StringVal("not b"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusFail,
									FailureMessages: []string{
										"A isn't.",
										// B doesn't fail because A is a precondition
										// and so pre-empts it.
									},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})
}

func TestContext2ApplySmokeTests_withData(t *testing.T) {
	cfg := testModule(t, "apply-smoketests-with-data")
	smokeTestAddr := addrs.SmokeTest{Name: "try"}
	smokeTestAddrConfig := smokeTestAddr.InModule(addrs.RootModule)
	smokeTestAddrAbs := smokeTestAddr.Absolute(addrs.RootModuleInstance)
	dataResourceAddr := addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "perchance",
		Name: "if_you_dont_mind",
	}
	dataResourceAddrConfig := dataResourceAddr.InModule(addrs.RootModule)
	//dataResourceAddrAbs := dataResourceAddr.Absolute(addrs.RootModuleInstance)
	providerAddr := addrs.MustParseProviderSourceString("example.com/test/perchance")

	if r := cfg.Module.ResourceByAddr(dataResourceAddr); r != nil {
		if r.SmokeTest == nil {
			t.Fatalf("config does not associate %s with any smoke test", dataResourceAddrConfig)
		}
		if r.SmokeTest.Name != smokeTestAddr.Name {
			t.Fatalf("config does not associate %s with %s", dataResourceAddrConfig, smokeTestAddrConfig)
		}
	} else {
		t.Fatalf("config does not include %s", dataResourceAddrConfig)
	}

	makePlan := func(t *testing.T, core *Context, aVal, bVal, cVal cty.Value) *plans.Plan {
		plan, diags := core.Plan(
			cfg, states.NewState(),
			SimplePlanOpts(plans.NormalMode, InputValues{
				"a": &InputValue{
					Value:      aVal,
					SourceType: ValueFromCaller,
				},
				"b": &InputValue{
					Value:      bVal,
					SourceType: ValueFromCaller,
				},
				"c": &InputValue{
					Value:      cVal,
					SourceType: ValueFromCaller,
				},
			}),
		)
		assertNoDiagnostics(t, diags)

		// We should know know that the smoke test exists as a checkable
		// object, but we don't know its status yet because we'll decide
		// that only during apply.
		got := plan.Checks
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusUnknown,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusUnknown,
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after plan\n%s", diff)
		}

		return plan
	}
	contextOpts := &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			providerAddr: func() (providers.Interface, error) {
				mock := &MockProvider{
					GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
						DataSources: map[string]providers.Schema{
							"perchance": {
								Block: &configschema.Block{
									Attributes: map[string]*configschema.Attribute{
										"b": {
											Type:     cty.String,
											Required: true,
										},
										"c": {
											Type:     cty.String,
											Required: true,
										},
										"splendid": {
											Type:     cty.Bool,
											Computed: true,
										},
									},
								},
							},
						},
					},
					ReadDataSourceFn: func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
						bVal := req.Config.GetAttr("b")
						cVal := req.Config.GetAttr("c")
						if !bVal.RawEquals(cty.StringVal("b")) {
							var diags tfdiags.Diagnostics
							diags = diags.Append(fmt.Errorf("B isn't."))
							return providers.ReadDataSourceResponse{
								Diagnostics: diags,
							}
						}
						splendid := cty.False
						if cVal.RawEquals(cty.StringVal("c")) {
							splendid = cty.True
						}
						return providers.ReadDataSourceResponse{
							State: cty.ObjectVal(map[string]cty.Value{
								"b":        bVal,
								"c":        cVal,
								"splendid": splendid,
							}),
						}
					},
				}
				return mock, nil
			},
		},
	}

	t.Run("all_passing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusPass,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusPass,
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("pre_failing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("not a"), cty.StringVal("b"), cty.StringVal("c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"A isn't."},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("data_failing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("not b"), cty.StringVal("c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"Failed to read data.perchance.if_you_dont_mind."},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("post_failing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("not c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"Rather frightful, actually."},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("all_failing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("not a"), cty.StringVal("not b"), cty.StringVal("not c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusFail,
									FailureMessages: []string{
										"A isn't.",
										// The others don't appear because the
										// precondition guards them.
									},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

	t.Run("data_and_post_failing", func(t *testing.T) {
		core := testContext2(t, contextOpts)
		plan := makePlan(t, core, cty.StringVal("a"), cty.StringVal("not b"), cty.StringVal("not c"))

		state, diags := core.Apply(plan, cfg)
		assertNoErrors(t, diags)

		// We should now have a final result for our object.
		got := state.CheckResults
		want := &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem(
					addrs.ConfigCheckable(smokeTestAddrConfig),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(
								addrs.Checkable(smokeTestAddrAbs),
								&states.CheckResultObject{
									Status: checks.StatusFail,
									FailureMessages: []string{
										"Failed to read data.perchance.if_you_dont_mind.",
										// The postcondition doesn't appear
										// because the data resource guards it.
									},
								},
							),
						),
					},
				),
			),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong check statuses after apply\n%s", diff)
		}
	})

}
