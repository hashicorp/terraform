package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

func TestPlan(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("plan")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_lockedState(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testPath := testFixturePath("plan")
	unlock, err := testLockState("./testdata", filepath.Join(testPath, DefaultStateFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	if err := os.Chdir(testPath); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{planPath}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestPlan_destroy(t *testing.T) {
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	outPath := testTempFile(t)
	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-destroy",
		"-out", outPath,
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}

	plan := testReadPlan(t, outPath)
	for _, m := range plan.Diff.Modules {
		for _, r := range m.Resources {
			if !r.Destroy {
				t.Fatalf("bad: %#v", r)
			}
		}
	}
}

func TestPlan_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that refresh was called
	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not be called")
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.NullVal(cty.EmptyObject)
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_outPath(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	td := testTempDir(t)
	outPath := filepath.Join(td, "test.plan")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.PlanResourceChangeResponse = providers.PlanResourceChangeResponse{
		PlannedState: cty.NullVal(cty.EmptyObject),
	}

	args := []string{
		"-out", outPath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if _, err := terraform.ReadPlan(f); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestPlan_outPathNoChange(t *testing.T) {
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
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	td := testTempDir(t)
	outPath := filepath.Join(td, "test.plan")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-out", outPath,
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	if !plan.Diff.Empty() {
		t.Fatalf("Expected empty plan to be written to plan file, got: %s", plan)
	}
}

// When using "-out" with a backend, the plan should encode the backend config
func TestPlan_outBackend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("plan-out-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Our state
	originalState := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	originalState.Init()

	// Setup our backend state
	dataState, srv := testBackendState(t, originalState, 200)
	defer srv.Close()
	testStateFileRemote(t, dataState)

	outPath := "foo"
	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-out", outPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	plan := testReadPlan(t, outPath)
	if !plan.Diff.Empty() {
		t.Fatalf("Expected empty plan to be written to plan file, got: %s", plan)
	}

	if plan.Backend.Empty() {
		t.Fatal("should have backend info")
	}
	if !reflect.DeepEqual(plan.Backend, dataState.Backend) {
		t.Fatalf("bad: %#v", plan.Backend)
	}
}

func TestPlan_refreshFalse(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-refresh=false",
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestPlan_state(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("bar"),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_stateDefault(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)

	// Change to that directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(filepath.Dir(statePath)); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		testFixturePath("plan"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("bar"),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_validate(t *testing.T) {
	// This is triggered by not asking for input so we have to set this to false
	test = false
	defer func() { test = true }()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("plan-invalid")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := ui.ErrorWriter.String()
	if !strings.Contains(actual, "cannot be computed") {
		t.Fatalf("bad: %s", actual)
	}
}

func TestPlan_vars(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return nil, nil
	}

	args := []string{
		"-var", "foo=bar",
		testFixturePath("plan-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varsUnset(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	defaultInputReader = bytes.NewBufferString("bar\n")

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		testFixturePath("plan-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_varFile(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return nil, nil
	}

	args := []string{
		"-var-file", varFilePath,
		testFixturePath("plan-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varFileDefault(t *testing.T) {
	varFileDir := testTempDir(t)
	varFilePath := filepath.Join(varFileDir, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(varFileDir); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return nil, nil
	}

	args := []string{
		testFixturePath("plan-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_detailedExitcode(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("plan")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{"-detailed-exitcode"}
	if code := c.Run(args); code != 2 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_detailedExitcode_emptyDiff(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("plan-emptydiff")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{"-detailed-exitcode"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestPlan_shutdown(t *testing.T) {
	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	p := testProvider()
	ui := new(cli.MockUi)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			ShutdownCh:       shutdownCh,
		},
	}

	p.StopFn = func() error {
		close(cancelled)
		return nil
	}

	var once sync.Once

	p.DiffFn = func(
		*terraform.InstanceInfo,
		*terraform.InstanceState,
		*terraform.ResourceConfig) (*terraform.InstanceDiff, error) {

		once.Do(func() {
			shutdownCh <- struct{}{}
		})

		// Because of the internal lock in the MockProvider, we can't
		// coordiante directly with the calling of Stop, and making the
		// MockProvider concurrent is disruptive to a lot of existing tests.
		// Wait here a moment to help make sure the main goroutine gets to the
		// Stop call before we exit, or the plan may finish before it can be
		// canceled.
		time.Sleep(200 * time.Millisecond)

		return &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if code := c.Run([]string{testFixturePath("apply-shutdown")}); code != 1 {
		// FIXME: we should be able to avoid the error during evaluation
		// the early exit isn't caught before the interpolation is evaluated
		t.Fatal(ui.OutputWriter.String())
	}

	select {
	case <-cancelled:
	default:
		t.Fatal("command not cancelled")
	}
}

const planVarFile = `
foo = "bar"
`

const testPlanNoStateStr = `
<not created>
`

const testPlanStateStr = `
ID = bar
Tainted = false
`

const testPlanStateDefaultStr = `
ID = bar
Tainted = false
`
