package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	backendinit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

func TestPlan(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_lockedState(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	unlock, err := testLockState(testDataDir, filepath.Join(td, DefaultStateFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
	}
}

func TestPlan_plan(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	planPath := testPlanFileNoop(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{planPath}
	if code := c.Run(args); code != 1 {
		t.Fatalf("wrong exit status %d; want 1\nstderr: %s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_destroy(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	outPath := testTempFile(t)
	statePath := testStateFile(t, originalState)

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-destroy",
		"-out", outPath,
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	for _, rc := range plan.Changes.Resources {
		if got, want := rc.Action, plans.Delete; got != want {
			t.Fatalf("wrong action %s for %s; want %s\nplanned change: %s", got, rc.Addr, want, spew.Sdump(rc))
		}
	}
}

func TestPlan_noState(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that refresh was called
	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not be called")
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.NullVal(p.GetProviderSchemaResponse.ResourceTypes["test_instance"].Block.ImpliedType())
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_outPath(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	outPath := filepath.Join(td, "test.plan")

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.NullVal(cty.EmptyObject),
	}

	args := []string{
		"-out", outPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	testReadPlan(t, outPath) // will call t.Fatal itself if the file cannot be read
}

func TestPlan_outPathNoChange(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// Aside from "id" (which is computed) the values here must
				// exactly match the values in the "plan" test fixture in order
				// to produce the empty plan we need for this test.
				AttrsJSON: []byte(`{"id":"bar","ami":"bar","network_interface":[{"description":"Main network interface","device_index":"0"}]}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, originalState)

	outPath := filepath.Join(td, "test.plan")

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-out", outPath,
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	if !plan.Changes.Empty() {
		t.Fatalf("Expected empty plan to be written to plan file, got: %s", spew.Sdump(plan))
	}
}

// When using "-out" with a backend, the plan should encode the backend config
func TestPlan_outBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-out-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","ami":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	// Set up our backend state
	dataState, srv := testBackendState(t, originalState, 200)
	defer srv.Close()
	testStateFileRemote(t, dataState)

	outPath := "foo"
	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"ami": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	ui := cli.NewMockUi()
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-out", outPath,
	}
	if code := c.Run(args); code != 0 {
		t.Logf("stdout: %s", ui.OutputWriter.String())
		t.Fatalf("plan command failed with exit code %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	if !plan.Changes.Empty() {
		t.Fatalf("Expected empty plan to be written to plan file, got: %s", spew.Sdump(plan))
	}

	if got, want := plan.Backend.Type, "http"; got != want {
		t.Errorf("wrong backend type %q; want %q", got, want)
	}
	if got, want := plan.Backend.Workspace, "default"; got != want {
		t.Errorf("wrong backend workspace %q; want %q", got, want)
	}
	{
		httpBackend := backendinit.Backend("http")()
		schema := httpBackend.ConfigSchema()
		got, err := plan.Backend.Config.Decode(schema.ImpliedType())
		if err != nil {
			t.Fatalf("failed to decode backend config in plan: %s", err)
		}
		want, err := dataState.Backend.Config(schema)
		if err != nil {
			t.Fatalf("failed to decode cached config: %s", err)
		}
		if !want.RawEquals(got) {
			t.Errorf("wrong backend config\ngot:  %#v\nwant: %#v", got, want)
		}
	}
}

func TestPlan_refreshFalse(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-refresh=false",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestPlan_state(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
		"network_interface": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
			"device_index": cty.String,
			"description":  cty.String,
		}))),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_stateDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Generate state and move it to the default path
	originalState := testState()
	statePath := testStateFile(t, originalState)
	os.Rename(statePath, path.Join(td, "terraform.tfstate"))

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
		"network_interface": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
			"device_index": cty.String,
			"description":  cty.String,
		}))),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_validate(t *testing.T) {
	// This is triggered by not asking for input so we have to set this to false
	test = false
	defer func() { test = true }()

	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-invalid"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := ui.ErrorWriter.String()
	if want := "Error: Invalid count argument"; !strings.Contains(actual, want) {
		t.Fatalf("unexpected error output\ngot:\n%s\n\nshould contain: %s", actual, want)
	}
}

func TestPlan_vars(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := planVarsFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{
		"-var", "foo=bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varsUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// The plan command will prompt for interactive input of var.foo.
	// We'll answer "bar" to that prompt, which should then allow this
	// configuration to apply even though var.foo doesn't have a
	// default value and there are no -var arguments on our command line.

	// This will (helpfully) panic if more than one variable is requested during plan:
	// https://github.com/hashicorp/terraform/issues/26027
	close := testInteractiveInput(t, []string{"bar"})
	defer close()

	p := planVarsFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

// This test adds a required argument to the test provider to validate
// processing of user input:
// https://github.com/hashicorp/terraform/issues/26035
func TestPlan_providerArgumentUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// The plan command will prompt for interactive input of provider.test.region
	defaultInputReader = bytes.NewBufferString("us-east-1\n")

	p := planFixtureProvider()
	// override the planFixtureProvider schema to include a required provider argument
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Required: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true, Computed: true},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_varFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{
		"-var-file", varFilePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varFileDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varFileWithDecls(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(planVarFileWithDecl), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	ui := cli.NewMockUi()
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-var-file", varFilePath,
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("succeeded; want failure\n\n%s", ui.OutputWriter.String())
	}

	msg := ui.ErrorWriter.String()
	if got, want := msg, "Variable declaration in .tfvars file"; !strings.Contains(got, want) {
		t.Fatalf("missing expected error message\nwant message containing %q\ngot:\n%s", want, got)
	}
}

func TestPlan_detailedExitcode(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := planFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-detailed-exitcode"}
	if code := c.Run(args); code != 2 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_detailedExitcode_emptyDiff(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan-emptydiff"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-detailed-exitcode"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_shutdown(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-shutdown"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
			ShutdownCh:       shutdownCh,
		},
	}

	p.StopFn = func() error {
		close(cancelled)
		return nil
	}

	var once sync.Once

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		once.Do(func() {
			shutdownCh <- struct{}{}
		})

		// Because of the internal lock in the MockProvider, we can't
		// coordinate directly with the calling of Stop, and making the
		// MockProvider concurrent is disruptive to a lot of existing tests.
		// Wait here a moment to help make sure the main goroutine gets to the
		// Stop call before we exit, or the plan may finish before it can be
		// canceled.
		time.Sleep(200 * time.Millisecond)

		s := req.ProposedNewState.AsValueMap()
		s["ami"] = cty.StringVal("bar")
		resp.PlannedState = cty.ObjectVal(s)
		return
	}

	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	code := c.Run([]string{})
	if code != 1 {
		t.Errorf("wrong exit code %d; want 1\noutput:\n%s", code, ui.OutputWriter.String())
	}

	select {
	case <-cancelled:
	default:
		t.Error("command not cancelled")
	}
}

func TestPlan_init_required(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("plan"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			// Running plan without setting testingOverrides is similar to plan without init
			Ui:   ui,
			View: view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("expected error, got success")
	}
	output := ui.ErrorWriter.String()
	if !strings.Contains(output, `Plugin reinitialization required. Please run "terraform init".`) {
		t.Fatal("wrong error message in output:", output)
	}
}

// Config with multiple resources, targeting plan of a subset
func TestPlan_targeted(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-targeted"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-target", "test_instance.foo",
		"-target", "test_instance.baz",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if got, want := done(t).Stdout(), "3 to add, 0 to change, 0 to destroy"; !strings.Contains(got, want) {
		t.Fatalf("bad change summary, want %q, got:\n%s", want, got)
	}
}

// Diagnostics for invalid -target flags
func TestPlan_targetFlagsDiags(t *testing.T) {
	testCases := map[string]string{
		"test_instance.": "Dot must be followed by attribute name.",
		"test_instance":  "Resource specification must include a resource type and name.",
	}

	for target, wantDiag := range testCases {
		t.Run(target, func(t *testing.T) {
			td := testTempDir(t)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			ui := new(cli.MockUi)
			view, _ := testView(t)
			c := &PlanCommand{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			}

			args := []string{
				"-target", target,
			}
			if code := c.Run(args); code != 1 {
				t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
			}

			got := ui.ErrorWriter.String()
			if !strings.Contains(got, target) {
				t.Fatalf("bad error output, want %q, got:\n%s", target, got)
			}
			if !strings.Contains(got, wantDiag) {
				t.Fatalf("bad error output, want %q, got:\n%s", wantDiag, got)
			}
		})
	}
}

// planFixtureSchema returns a schema suitable for processing the
// configuration in testdata/plan . This schema should be
// assigned to a mock provider named "test".
func planFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

// planFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/plan. This mock has
// GetSchemaResponse and PlanResourceChangeFn populated, with the plan
// step just passing through the new object proposed by Terraform Core.
func planFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = planFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	return p
}

// planVarsFixtureSchema returns a schema suitable for processing the
// configuration in testdata/plan-vars . This schema should be
// assigned to a mock provider named "test".
func planVarsFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

// planVarsFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/plan-vars. This mock has
// GetSchemaResponse and PlanResourceChangeFn populated, with the plan
// step just passing through the new object proposed by Terraform Core.
func planVarsFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = planVarsFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	return p
}

const planVarFile = `
foo = "bar"
`

const planVarFileWithDecl = `
foo = "bar"

variable "nope" {
}
`
