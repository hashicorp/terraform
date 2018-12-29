package command

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestShow(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_JSONStateNotImplemented(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	defer testChdir(t, filepath.Dir(statePath))()
	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-json",
		statePath,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_noArgs(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	defer testChdir(t, filepath.Dir(statePath))()

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_noArgsNoState(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	defer testChdir(t, filepath.Dir(statePath))()

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_plan(t *testing.T) {
	planPath := testPlanFileNoop(t)

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestShow_plan_json(t *testing.T) {
	planPath := showFixturePlanFile(t)

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-json",
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestShow_state(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)

	ui := new(cli.MockUi)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// showFixtureSchema returns a schema suitable for processing the configuration
// in test-fixtures/show. This schema should be assigned to a mock provider
// named "test".
func showFixtureSchema() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}
}

// showFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in test-fixtures/show. This mock has
// GetSchemaReturn, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func showFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetSchemaReturn = showFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: cty.UnknownAsNull(req.PlannedState),
		}
	}
	return p
}

// showFixturePlanFile creates a plan file at a temporary location containing a
// single change to create the test_instance.foo that is included in the "show"
// test fixture, returning the location of that plan file.
func showFixturePlanFile(t *testing.T) string {
	_, snap := testModuleWithSnapshot(t, "show")
	plannedVal := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.UnknownVal(cty.String),
		"ami": cty.StringVal("bar"),
	})
	priorValRaw, err := plans.NewDynamicValue(cty.NullVal(plannedVal.Type()), plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plannedValRaw, err := plans.NewDynamicValue(plannedVal, plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := testPlan(t)
	plan.Changes.SyncWrapper().AppendResourceInstanceChange(&plans.ResourceInstanceChangeSrc{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		ProviderAddr: addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Create,
			Before: priorValRaw,
			After:  plannedValRaw,
		},
	})
	return testPlanFile(
		t,
		snap,
		states.NewState(),
		plan,
	)
}
