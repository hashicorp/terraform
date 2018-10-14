package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

func TestApply(t *testing.T) {
	statePath := testTempFile(t)

	p := applyFixtureProvider()

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

// test apply with locked state
func TestApply_lockedState(t *testing.T) {
	statePath := testTempFile(t)

	unlock, err := testLockState("./testdata", statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
	}
}

// test apply with locked state, waiting for unlock
func TestApply_lockedStateWait(t *testing.T) {
	statePath := testTempFile(t)

	unlock, err := testLockState("./testdata", statePath)
	if err != nil {
		t.Fatal(err)
	}

	// unlock during apply
	go func() {
		time.Sleep(500 * time.Millisecond)
		unlock()
	}()

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// wait 4s just in case the lock process doesn't release in under a second,
	// and we want our context to be alive for a second retry at the 3s mark.
	args := []string{
		"-state", statePath,
		"-lock-timeout", "4s",
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("lock should have succeeded in less than 3s: %s", ui.ErrorWriter)
	}
}

// high water mark counter
type hwm struct {
	sync.Mutex
	val int
	max int
}

func (t *hwm) Inc() {
	t.Lock()
	defer t.Unlock()
	t.val++
	if t.val > t.max {
		t.max = t.val
	}
}

func (t *hwm) Dec() {
	t.Lock()
	defer t.Unlock()
	t.val--
}

func (t *hwm) Max() int {
	t.Lock()
	defer t.Unlock()
	return t.max
}

func TestApply_parallelism(t *testing.T) {
	statePath := testTempFile(t)

	par := 4

	// This blocks all the apply functions. We close it when we exit so
	// they end quickly after this test finishes.
	block := make(chan struct{})
	// signal how many goroutines have started
	started := make(chan int, 100)

	runCount := &hwm{}

	// Since our mock provider has its own mutex preventing concurrent calls
	// to ApplyResourceChange, we need to use a number of separate providers
	// here. They will all have the same mock implementation function assigned
	// but crucially they will each have their own mutex.
	providerFactories := map[string]providers.Factory{}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("test%d", i)
		provider := &terraform.MockProvider{}
		provider.GetSchemaReturn = &terraform.ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				name + "_instance": {},
			},
		}
		provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			// Increment so we're counting parallelism
			started <- 1
			runCount.Inc()
			defer runCount.Dec()
			// Block here to stage up our max number of parallel instances
			<-block

			return providers.ApplyResourceChangeResponse{
				NewState: cty.EmptyObjectVal,
			}
		}
		providerFactories[name] = providers.FactoryFixed(provider)
	}
	testingOverrides := &testingOverrides{
		ProviderResolver: providers.ResolverFixed(providerFactories),
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: testingOverrides,
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		fmt.Sprintf("-parallelism=%d", par),
		testFixturePath("parallelism"),
	}

	// Run in a goroutine. We can get any errors from the ui.OutputWriter
	doneCh := make(chan int, 1)
	go func() {
		doneCh <- c.Run(args)
	}()

	timeout := time.After(5 * time.Second)

	// ensure things are running
	for i := 0; i < par; i++ {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for all goroutines to start")
		case <-started:
		}
	}

	// a little extra sleep, since we can't ensure all goroutines from the walk have
	// really started
	time.Sleep(100 * time.Millisecond)
	close(block)

	select {
	case res := <-doneCh:
		if res != 0 {
			t.Fatal(ui.OutputWriter.String())
		}
	case <-timeout:
		t.Fatal("timeout waiting from Run()")
	}

	// The total in flight should equal the parallelism
	if runCount.Max() != par {
		t.Fatalf("Expected parallelism: %d, got: %d", par, runCount.Max())
	}
}

func TestApply_configInvalid(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", testTempFile(t),
		"-auto-approve",
		testFixturePath("apply-config-invalid"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_defaultState(t *testing.T) {
	td := testTempDir(t)
	statePath := filepath.Join(td, DefaultStateFilename)

	// Change to the temporary directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(filepath.Dir(statePath)); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// create an existing state file
	localState := &state.LocalState{Path: statePath}
	if err := localState.WriteState(terraform.NewState()); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_error(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	var lock sync.Mutex
	errored := false
	p.ApplyFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		lock.Lock()
		defer lock.Unlock()

		if !errored {
			errored = true
			return nil, fmt.Errorf("error")
		}

		return &terraform.InstanceState{ID: "foo"}, nil
	}
	p.DiffFn = func(
		*terraform.InstanceInfo,
		*terraform.InstanceState,
		*terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		return &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":    {Type: cty.String, Optional: true, Computed: true},
					"ami":   {Type: cty.String, Optional: true},
					"error": {Type: cty.Bool, Optional: true},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply-error"),
	}
	if ui.ErrorWriter != nil {
		t.Logf("stdout:\n%s", ui.OutputWriter.String())
		t.Logf("stderr:\n%s", ui.ErrorWriter.String())
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("wrong exit code %d; want 1", code)
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if len(state.RootModule().Resources) == 0 {
		t.Fatal("no resources in state")
	}
}

func TestApply_input(t *testing.T) {
	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Set some default reader/writers for the inputs
	defaultInputReader = bytes.NewBufferString("foo\n")
	defaultInputWriter = new(bytes.Buffer)

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply-input"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expected := strings.TrimSpace(`
<no state>
Outputs:

result = foo
	`)
	testStateOutput(t, statePath, expected)
}

// When only a partial set of the variables are set, Terraform
// should still ask for the unset ones by default (with -input=true)
func TestApply_inputPartial(t *testing.T) {
	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Set some default reader/writers for the inputs
	defaultInputReader = bytes.NewBufferString("one\ntwo\n")
	defaultInputWriter = new(bytes.Buffer)

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-var", "foo=foovalue",
		testFixturePath("apply-input-partial"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expected := strings.TrimSpace(`
<no state>
Outputs:

bar = one
foo = foovalue
	`)
	testStateOutput(t, statePath, expected)
}

func TestApply_noArgs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("apply")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	statePath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_plan(t *testing.T) {
	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Set some default reader/writers for the inputs
	defaultInputReader = new(bytes.Buffer)
	defaultInputWriter = new(bytes.Buffer)

	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state-out", statePath,
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_plan_backup(t *testing.T) {
	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)
	backupPath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// create a state file that needs to be backed up
	err := statemgr.NewFilesystem(statePath).WriteState(states.NewState())
	if err != nil {
		t.Fatal(err)
	}
	args := []string{
		"-state-out", statePath,
		"-backup", backupPath,
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Should have a backup file
	testStateRead(t, backupPath)
}

func TestApply_plan_noBackup(t *testing.T) {
	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state-out", statePath,
		"-backup", "-",
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Ensure there is no backup
	_, err := os.Stat(statePath + DefaultBackupExtension)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}

	// Ensure there is no literal "-"
	_, err = os.Stat("-")
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
}

func TestApply_plan_remoteState(t *testing.T) {
	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)
	remoteStatePath := filepath.Join(tmp, DefaultDataDir, DefaultStateFilename)
	if err := os.MkdirAll(filepath.Dir(remoteStatePath), 0755); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Set some default reader/writers for the inputs
	defaultInputReader = new(bytes.Buffer)
	defaultInputWriter = new(bytes.Buffer)

	// Create a remote state
	state := testState()
	backendState, srv := testRemoteState(t, state, 200)
	defer srv.Close()
	testStateFileRemote(t, backendState)

	_, snap := testModuleWithSnapshot(t, "apply")
	backendConfig := cty.ObjectVal(map[string]cty.Value{
		"address":                cty.StringVal(srv.URL),
		"update_method":          cty.NullVal(cty.String),
		"lock_address":           cty.NullVal(cty.String),
		"unlock_address":         cty.NullVal(cty.String),
		"lock_method":            cty.NullVal(cty.String),
		"unlock_method":          cty.NullVal(cty.String),
		"username":               cty.NullVal(cty.String),
		"password":               cty.NullVal(cty.String),
		"skip_cert_verification": cty.NullVal(cty.Bool),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfig, backendConfig.Type())
	if err != nil {
		t.Fatal(err)
	}
	planPath := testPlanFile(t, snap, state, &plans.Plan{
		Backend: plans.Backend{
			Type:   "http",
			Config: backendConfigRaw,
		},
		Changes: plans.NewChanges(),
	})

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// State file should be not be installed
	if _, err := os.Stat(filepath.Join(tmp, DefaultStateFilename)); err == nil {
		data, _ := ioutil.ReadFile(DefaultStateFilename)
		t.Fatalf("State path should not exist: %s", string(data))
	}

	// Check that there is no remote state config
	if _, err := os.Stat(remoteStatePath); err == nil {
		t.Fatalf("has remote state config")
	}
}

func TestApply_planWithVarFile(t *testing.T) {
	varFileDir := testTempDir(t)
	varFilePath := filepath.Join(varFileDir, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(varFileDir); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state-out", statePath,
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_planVars(t *testing.T) {
	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-var", "foo=bar",
		planPath,
	}
	if code := c.Run(args); code == 0 {
		t.Fatal("should've failed")
	}
}

// we should be able to apply a plan file with no other file dependencies
func TestApply_planNoModuleFiles(t *testing.T) {
	// temporary data directory which we can remove between commands
	td := testTempDir(t)
	defer os.RemoveAll(td)

	defer testChdir(t, td)()

	p := applyFixtureProvider()
	planPath := applyFixturePlanFile(t)

	apply := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               new(cli.MockUi),
		},
	}
	args := []string{
		planPath,
	}
	apply.Run(args)
	if p.ValidateProviderConfigCalled {
		t.Fatal("Validate should not be called with a plan")
	}
}

func TestApply_refresh(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"ami":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ReadResourceCalled {
		t.Fatal("should call ReadResource")
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Should have a backup file
	backupState := testStateRead(t, statePath+DefaultBackupExtension)

	actualStr := strings.TrimSpace(backupState.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_shutdown(t *testing.T) {
	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	statePath := testTempFile(t)
	p := testProvider()

	ui := new(cli.MockUi)
	c := &ApplyCommand{
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

	p.DiffFn = func(
		*terraform.InstanceInfo,
		*terraform.InstanceState,
		*terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		return &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	var once sync.Once
	p.ApplyFn = func(
		*terraform.InstanceInfo,
		*terraform.InstanceState,
		*terraform.InstanceDiff) (*terraform.InstanceState, error) {

		// only cancel once
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

		return &terraform.InstanceState{
			ID: "foo",
			Attributes: map[string]string{
				"ami": "2",
			},
		}, nil
	}

	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply-shutdown"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	select {
	case <-cancelled:
	default:
		t.Fatal("command not cancelled")
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_state(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"ami":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}
	p.ApplyResourceChangeResponse = providers.ApplyResourceChangeResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.NullVal(cty.String),
		"ami": cty.StringVal("bar"),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state during plan\ngot: %#v\nwant: %#v", actual, expected)
	}

	actual = p.ApplyResourceChangeRequest.PriorState
	expected = cty.ObjectVal(map[string]cty.Value{
		"id":  cty.NullVal(cty.String),
		"ami": cty.StringVal("bar"),
	})
	if actual != expected {
		t.Fatalf("wrong prior state during apply\ngot: %#v\nwant: %#v", actual, expected)
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	backupState := testStateRead(t, statePath+DefaultBackupExtension)

	actualStr := strings.TrimSpace(backupState.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_stateNoExist(t *testing.T) {
	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"idontexist.tfstate",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_sensitiveOutput(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	statePath := testTempFile(t)

	args := []string{
		"-state", statePath,
		"-auto-approve",
		testFixturePath("apply-sensitive-output"),
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "notsensitive = Hello world") {
		t.Fatalf("bad: output should contain 'notsensitive' output\n%s", output)
	}
	if !strings.Contains(output, "sensitive = <sensitive>") {
		t.Fatalf("bad: output should contain 'sensitive' output\n%s", output)
	}
}

func TestApply_vars(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.InstanceDiff{}, nil
	}

	args := []string{
		"-auto-approve",
		"-var", "foo=bar",
		"-state", statePath,
		testFixturePath("apply-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFile(t *testing.T) {
	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.InstanceDiff{}, nil
	}

	args := []string{
		"-auto-approve",
		"-var-file", varFilePath,
		"-state", statePath,
		testFixturePath("apply-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefault(t *testing.T) {
	varFileDir := testTempDir(t)
	varFilePath := filepath.Join(varFileDir, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

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
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.InstanceDiff{}, nil
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
		testFixturePath("apply-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefaultJSON(t *testing.T) {
	varFileDir := testTempDir(t)
	varFilePath := filepath.Join(varFileDir, "terraform.tfvars.json")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFileJSON), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

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
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}

	actual := ""
	p.DiffFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.InstanceDiff{}, nil
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
		testFixturePath("apply-vars"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_backup(t *testing.T) {
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte("{\n            \"id\": \"bar\"\n          }"),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
		)
	})
	statePath := testStateFile(t, originalState)
	backupPath := testTempFile(t)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", backupPath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	backupState := testStateRead(t, backupPath)

	actual := backupState.RootModule().Resources["test_instance.foo"]
	expected := originalState.RootModule().Resources["test_instance.foo"]
	if !cmp.Equal(actual, expected) {
		t.Fatalf(
			"wrong aws_instance.foo state\n%s",
			cmp.Diff(expected, actual, cmp.Transformer("bytesAsString", func(b []byte) string {
				return string(b)
			})),
		)
	}
}

func TestApply_disableBackup(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", "-",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state during plan\ngot:  %#v\nwant: %#v", actual, expected)
	}

	actual = p.ApplyResourceChangeRequest.PriorState
	expected = cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state during apply\ngot:  %#v\nwant: %#v", actual, expected)
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Ensure there is no backup
	_, err := os.Stat(statePath + DefaultBackupExtension)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}

	// Ensure there is no literal "-"
	_, err = os.Stat("-")
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
}

// Test that the Terraform env is passed through
func TestApply_terraformEnv(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
		testFixturePath("apply-terraform-env"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expected := strings.TrimSpace(`
<no state>
Outputs:

output = default
	`)
	testStateOutput(t, statePath, expected)
}

// Test that the Terraform env is passed through
func TestApply_terraformEnvNonDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create new env
	{
		ui := new(cli.MockUi)
		newCmd := &WorkspaceNewCommand{}
		newCmd.Meta = Meta{Ui: ui}
		if code := newCmd.Run([]string{"test"}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	// Switch to it
	{
		args := []string{"test"}
		ui := new(cli.MockUi)
		selCmd := &WorkspaceSelectCommand{}
		selCmd.Meta = Meta{Ui: ui}
		if code := selCmd.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{
		"-auto-approve",
		testFixturePath("apply-terraform-env"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	statePath := filepath.Join("terraform.tfstate.d", "test", "terraform.tfstate")
	expected := strings.TrimSpace(`
<no state>
Outputs:

output = test
	`)
	testStateOutput(t, statePath, expected)
}

func testHttpServer(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/header", testHttpHandlerHeader)

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpHandlerHeader(w http.ResponseWriter, r *http.Request) {
	var url url.URL
	url.Scheme = "file"
	url.Path = filepath.ToSlash(testFixturePath("init"))

	w.Header().Add("X-Terraform-Get", url.String())
	w.WriteHeader(200)
}

// applyFixtureSchema returns a schema suitable for processing the
// configuration in test-fixtures/apply . This schema should be
// assigned to a mock provider named "test".
func applyFixtureSchema() *terraform.ProviderSchema {
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

// applyFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in test-fixtures/apply. This mock has
// GetSchemaReturn, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func applyFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetSchemaReturn = applyFixtureSchema()
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

// applyFixturePlanFile creates a plan file at a temporary location containing
// a single change to create the test_instance.foo that is included in the
// "apply" test fixture, returning the location of that plan file.
func applyFixturePlanFile(t *testing.T) string {
	_, snap := testModuleWithSnapshot(t, "apply")
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

const applyVarFile = `
foo = "bar"
`

const applyVarFileJSON = `
{ "foo": "bar" }
`

const testApplyDisableBackupStr = `
ID = bar
Tainted = false
`

const testApplyDisableBackupStateStr = `
ID = bar
Tainted = false
`

const testApplyStateStr = `
ID = bar
Tainted = false
`

const testApplyStateDiffStr = `
ID = bar
Tainted = false
`
