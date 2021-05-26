package command

import (
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestApply_destroy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
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
	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(stateFile.State.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr = strings.TrimSpace(backupStateFile.State.String())
	expectedStr = strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyApproveNo(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create some existing state
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
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	defer testInputMap(t, map[string]string{
		"approve": "no",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if got, want := output.Stdout(), "Destroy cancelled"; !strings.Contains(got, want) {
		t.Fatalf("expected output to include %q, but was:\n%s", want, got)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyApproveYes(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create some existing state
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
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	defer testInputMap(t, map[string]string{
		"approve": "yes",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyLockedState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
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
	statePath := testStateFile(t, originalState)

	unlock, err := testLockState(testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
	}

	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	if !strings.Contains(output.Stderr(), "lock") {
		t.Fatal("command output does not look like a lock error:", output.Stderr())
	}
}

func TestApply_destroyPlan(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	planPath := testPlanFileNoop(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if !strings.Contains(output.Stderr(), "plan file") {
		t.Fatal("expected command output to refer to plan file, but got:", output.Stderr())
	}
}

func TestApply_destroyPath(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		testFixturePath("apply"),
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if !strings.Contains(output.Stderr(), "-chdir") {
		t.Fatal("expected command output to refer to -chdir flag, but got:", output.Stderr())
	}
}

// Config with multiple resources with dependencies, targeting destroy of a
// root node, expecting all other resources to be destroyed due to
// dependencies.
func TestApply_destroyTargetedDependencies(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-destroy-targeted"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"i-ab123"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_load_balancer",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"i-abc123"}`),
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
				Status:       states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, originalState)

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
			"test_load_balancer": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":        {Type: cty.String, Computed: true},
						"instances": {Type: cty.List(cty.String), Optional: true},
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

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-target", "test_instance.foo",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile == nil || stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	spew.Config.DisableMethods = true
	if !stateFile.State.Empty() {
		t.Fatalf("unexpected final state\ngot: %s\nwant: empty state", spew.Sdump(stateFile.State))
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr := strings.TrimSpace(backupStateFile.State.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", actualStr, expectedStr)
	}
}

// Config with multiple resources with dependencies, targeting destroy of a
// leaf node, expecting the other resources to remain.
func TestApply_destroyTargeted(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-destroy-targeted"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"i-ab123"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_load_balancer",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"i-abc123"}`),
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
				Status:       states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	wantState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"i-ab123"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, originalState)

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
			"test_load_balancer": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":        {Type: cty.String, Computed: true},
						"instances": {Type: cty.List(cty.String), Optional: true},
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

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-target", "test_load_balancer.foo",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile == nil || stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(stateFile.State.String())
	expectedStr := strings.TrimSpace(wantState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupActualStr := strings.TrimSpace(backupStateFile.State.String())
	backupExpectedStr := strings.TrimSpace(originalState.String())
	if backupActualStr != backupExpectedStr {
		t.Fatalf("bad:\n\nactual:\n%s\n\nexpected:\nb%s", backupActualStr, backupExpectedStr)
	}
}

const testApplyDestroyStr = `
<no state>
`
