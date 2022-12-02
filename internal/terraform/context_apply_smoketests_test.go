package terraform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
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
