package command

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/copy"
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

func TestShow_json_output(t *testing.T) {
	fixtureDir := "testdata/show-json"
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			td := tempDir(t)
			inputDir := filepath.Join(fixtureDir, entry.Name())
			copy.CopyDir(inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			p := showFixtureProvider()
			ui := new(cli.MockUi)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			}

			// init
			ic := &InitCommand{
				Meta: m,
				providerInstaller: &mockProviderInstaller{
					Providers: map[string][]string{
						"test": []string{"1.2.3"},
					},
					Dir: m.pluginDir(),
				},
			}
			if code := ic.Run([]string{}); code != 0 {
				t.Fatalf("init failed\n%s", ui.ErrorWriter)
			}

			pc := &PlanCommand{
				Meta: m,
			}

			args := []string{
				"-out=terraform.plan",
			}

			if code := pc.Run(args); code != 0 {
				t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
			}

			// flush the plan output from the mock ui
			ui.OutputWriter.Reset()
			sc := &ShowCommand{
				Meta: m,
			}

			args = []string{
				"-json",
				"terraform.plan",
			}
			defer os.Remove("terraform.plan")

			if code := sc.Run(args); code != 0 {
				t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
			}

			// compare ui output to wanted output
			var got, want plan

			gotString := ui.OutputWriter.String()
			json.Unmarshal([]byte(gotString), &got)

			wantFile, err := os.Open("output.json")
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			defer wantFile.Close()
			byteValue, err := ioutil.ReadAll(wantFile)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			json.Unmarshal([]byte(byteValue), &want)

			if !cmp.Equal(got, want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
			}

		})

	}
}

// similar test as above, without the plan
func TestShow_json_output_state(t *testing.T) {
	fixtureDir := "testdata/show-json-state"
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			td := tempDir(t)
			inputDir := filepath.Join(fixtureDir, entry.Name())
			copy.CopyDir(inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			p := showFixtureProvider()
			ui := new(cli.MockUi)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			}

			// init
			ic := &InitCommand{
				Meta: m,
				providerInstaller: &mockProviderInstaller{
					Providers: map[string][]string{
						"test": []string{"1.2.3"},
					},
					Dir: m.pluginDir(),
				},
			}
			if code := ic.Run([]string{}); code != 0 {
				t.Fatalf("init failed\n%s", ui.ErrorWriter)
			}

			// flush the plan output from the mock ui
			ui.OutputWriter.Reset()
			sc := &ShowCommand{
				Meta: m,
			}

			if code := sc.Run([]string{"-json"}); code != 0 {
				t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
			}

			// compare ui output to wanted output
			type state struct {
				FormatVersion    string                 `json:"format_version,omitempty"`
				TerraformVersion string                 `json:"terraform_version"`
				Values           map[string]interface{} `json:"values,omitempty"`
			}
			var got, want state

			gotString := ui.OutputWriter.String()
			json.Unmarshal([]byte(gotString), &got)

			wantFile, err := os.Open("output.json")
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			defer wantFile.Close()
			byteValue, err := ioutil.ReadAll(wantFile)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			json.Unmarshal([]byte(byteValue), &want)

			if !cmp.Equal(got, want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
			}

		})

	}
}

// showFixtureSchema returns a schema suitable for processing the configuration
// in testdata/show. This schema should be assigned to a mock provider
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
// operation with the configuration in testdata/show. This mock has
// GetSchemaReturn, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func showFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetSchemaReturn = showFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		idVal := req.ProposedNewState.GetAttr("id")
		amiVal := req.ProposedNewState.GetAttr("ami")
		if idVal.IsNull() {
			idVal = cty.UnknownVal(cty.String)
		}
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"id":  idVal,
				"ami": amiVal,
			}),
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		idVal := req.PlannedState.GetAttr("id")
		amiVal := req.PlannedState.GetAttr("ami")
		if !idVal.IsKnown() {
			idVal = cty.StringVal("placeholder")
		}
		return providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"id":  idVal,
				"ami": amiVal,
			}),
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

// this simplified plan struct allows us to preserve field order when marshaling
// the command output. NOTE: we are leaving "terraform_version" out of this test
// to avoid needing to constantly update the expected output; as a potential
// TODO we could write a jsonplan compare function.
type plan struct {
	FormatVersion   string                 `json:"format_version,omitempty"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
	PlannedValues   map[string]interface{} `json:"planned_values,omitempty"`
	ResourceChanges []interface{}          `json:"resource_changes,omitempty"`
	OutputChanges   map[string]interface{} `json:"output_changes,omitempty"`
	PriorState      priorState             `json:"prior_state,omitempty"`
	Config          map[string]interface{} `json:"configuration,omitempty"`
}

type priorState struct {
	FormatVersion string                 `json:"format_version,omitempty"`
	Values        map[string]interface{} `json:"values,omitempty"`
}
