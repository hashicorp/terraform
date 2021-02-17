package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
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
	stateDir := filepath.Dir(statePath)
	defer os.RemoveAll(stateDir)
	defer testChdir(t, stateDir)()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// the statefile created by testStateFile is named state.tfstate
	// so one arg is required
	args := []string{"state.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}

	if !strings.Contains(ui.OutputWriter.String(), "# test_instance.foo:") {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// https://github.com/hashicorp/terraform/issues/21462
func TestShow_aliasedProvider(t *testing.T) {
	// Create the default state with aliased resource
	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// The weird whitespace here is reflective of how this would
				// get written out in a real state file, due to the indentation
				// of all of the containing wrapping objects and arrays.
				AttrsJSON:    []byte("{\n            \"id\": \"bar\"\n          }"),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{},
			},
			addrs.RootModuleInstance.ProviderConfigAliased(addrs.NewDefaultProvider("test"), "alias"),
		)
	})

	statePath := testStateFile(t, testState)
	stateDir := filepath.Dir(statePath)
	defer os.RemoveAll(stateDir)
	defer testChdir(t, stateDir)()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	fmt.Println(os.Getwd())

	// the statefile created by testStateFile is named state.tfstate
	args := []string{"state.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad exit code: \n%s", ui.OutputWriter.String())
	}

	if strings.Contains(ui.OutputWriter.String(), "# missing schema for provider \"test.alias\"") {
		t.Fatalf("bad output: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_noArgsNoState(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	stateDir := filepath.Dir(statePath)
	defer os.RemoveAll(stateDir)
	defer testChdir(t, stateDir)()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// the statefile created by testStateFile is named state.tfstate
	args := []string{"state.tfstate"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestShow_plan(t *testing.T) {
	planPath := testPlanFileNoop(t)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	want := `Terraform will perform the following actions`
	got := done(t).Stdout()
	if !strings.Contains(got, want) {
		t.Errorf("missing expected output\nwant: %s\ngot:\n%s", want, got)
	}
}

func TestShow_planWithChanges(t *testing.T) {
	planPathWithChanges := showFixturePlanFile(t, plans.DeleteThenCreate)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		planPathWithChanges,
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	want := `test_instance.foo must be replaced`
	got := done(t).Stdout()
	if !strings.Contains(got, want) {
		t.Errorf("missing expected output\nwant: %s\ngot:\n%s", want, got)
	}
}

func TestShow_plan_json(t *testing.T) {
	planPath := showFixturePlanFile(t, plans.Create)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			Ui:               ui,
			View:             view,
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
	defer os.RemoveAll(filepath.Dir(statePath))

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
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
			testCopyDir(t, inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			expectError := strings.Contains(entry.Name(), "error")

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": []string{"1.2.3"},
			})
			defer close()

			p := showFixtureProvider()
			ui := new(cli.MockUi)
			view, _ := testView(t)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
				ProviderSource:   providerSource,
			}

			// init
			ic := &InitCommand{
				Meta: m,
			}
			if code := ic.Run([]string{}); code != 0 {
				if expectError {
					// this should error, but not panic.
					return
				}
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
			testCopyDir(t, inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": []string{"1.2.3"},
			})
			defer close()

			p := showFixtureProvider()
			ui := new(cli.MockUi)
			view, _ := testView(t)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
				ProviderSource:   providerSource,
			}

			// init
			ic := &InitCommand{
				Meta: m,
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
func showFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

// showFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/show. This mock has
// GetSchemaResponse, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func showFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = showFixtureSchema()
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
// single change to create or update the test_instance.foo that is included in the "show"
// test fixture, returning the location of that plan file.
// `action` is the planned change you would like to elicit
func showFixturePlanFile(t *testing.T, action plans.Action) string {
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
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		ChangeSrc: plans.ChangeSrc{
			Action: action,
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
