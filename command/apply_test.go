package command

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

func TestApply(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_path(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "-chdir") {
		t.Fatal("expected command output to refer to -chdir flag, but got:", output)
	}
}

func TestApply_approveNo(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Answer approval request with "no"
	defaultInputReader = bytes.NewBufferString("no\n")
	defaultInputWriter = new(bytes.Buffer)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if got, want := done(t).Stdout(), "Apply cancelled"; !strings.Contains(got, want) {
		t.Fatalf("expected output to include %q, but was:\n%s", want, got)
	}

	if _, err := os.Stat(statePath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("state file should not exist")
	}
}

func TestApply_approveYes(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Answer approval request with "yes"
	defaultInputReader = bytes.NewBufferString("yes\n")
	defaultInputWriter = new(bytes.Buffer)

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	unlock, err := testLockState(testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	unlock, err := testLockState(testDataDir, statePath)
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// wait 4s just in case the lock process doesn't release in under a second,
	// and we want our context to be alive for a second retry at the 3s mark.
	args := []string{
		"-state", statePath,
		"-lock-timeout", "4s",
		"-auto-approve",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("lock should have succeeded in less than 3s: %s", ui.ErrorWriter)
	}
}

// Verify that the parallelism flag allows no more than the desired number of
// concurrent calls to ApplyResourceChange.
func TestApply_parallelism(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("parallelism"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	par := 4

	// started is a semaphore that we use to ensure that we never have more
	// than "par" apply operations happening concurrently
	started := make(chan struct{}, par)

	// beginCtx is used as a starting gate to hold back ApplyResourceChange
	// calls until we reach the desired concurrency. The cancel func "begin" is
	// called once we reach the desired concurrency, allowing all apply calls
	// to proceed in unison.
	beginCtx, begin := context.WithCancel(context.Background())

	// Since our mock provider has its own mutex preventing concurrent calls
	// to ApplyResourceChange, we need to use a number of separate providers
	// here. They will all have the same mock implementation function assigned
	// but crucially they will each have their own mutex.
	providerFactories := map[addrs.Provider]providers.Factory{}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("test%d", i)
		provider := &terraform.MockProvider{}
		provider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				name + "_instance": {Block: &configschema.Block{}},
			},
		}
		provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {

			// If we ever have more than our intended parallelism number of
			// apply operations running concurrently, the semaphore will fail.
			select {
			case started <- struct{}{}:
				defer func() {
					<-started
				}()
			default:
				t.Fatal("too many concurrent apply operations")
			}

			// If we never reach our intended parallelism, the context will
			// never be canceled and the test will time out.
			if len(started) >= par {
				begin()
			}
			<-beginCtx.Done()

			// do some "work"
			// Not required for correctness, but makes it easier to spot a
			// failure when there is more overlap.
			time.Sleep(10 * time.Millisecond)

			return providers.ApplyResourceChangeResponse{
				NewState: cty.EmptyObjectVal,
			}
		}
		providerFactories[addrs.NewDefaultProvider(name)] = providers.FactoryFixed(provider)
	}
	testingOverrides := &testingOverrides{
		Providers: providerFactories,
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: testingOverrides,
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		fmt.Sprintf("-parallelism=%d", par),
	}

	res := c.Run(args)
	if res != 0 {
		t.Fatal(ui.OutputWriter.String())
	}
}

func TestApply_configInvalid(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-config-invalid"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", testTempFile(t),
		"-auto-approve",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_defaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// create an existing state file
	localState := statemgr.NewFilesystem(statePath)
	if err := localState.WriteState(states.NewState()); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"-auto-approve",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-error"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	ui := cli.NewMockUi()
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	var lock sync.Mutex
	errored := false
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		lock.Lock()
		defer lock.Unlock()

		if !errored {
			errored = true
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error"))
		}

		s := req.PlannedState.AsValueMap()
		s["id"] = cty.StringVal("foo")

		resp.NewState = cty.ObjectVal(s)
		return
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		s := req.ProposedNewState.AsValueMap()
		s["id"] = cty.UnknownVal(cty.String)
		resp.PlannedState = cty.ObjectVal(s)
		return
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true},
						"error": {Type: cty.Bool, Optional: true},
					},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-input"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// The configuration for this test includes a declaration of variable
	// "foo" with no default, and we don't set it on the command line below,
	// so the apply command will produce an interactive prompt for the
	// value of var.foo. We'll answer "foo" here, and we expect the output
	// value "result" to echo that back to us below.
	defaultInputReader = bytes.NewBufferString("foo\n")
	defaultInputWriter = new(bytes.Buffer)

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-input-partial"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Set some default reader/writers for the inputs
	defaultInputReader = bytes.NewBufferString("one\ntwo\n")
	defaultInputWriter = new(bytes.Buffer)

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-var", "foo=foovalue",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// create a state file that needs to be backed up
	err := statemgr.NewFilesystem(statePath).WriteState(states.NewState())
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"-state", statePath,
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	_, srv := testRemoteState(t, state, 200)
	defer srv.Close()

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
		"retry_max":              cty.NullVal(cty.String),
		"retry_wait_min":         cty.NullVal(cty.String),
		"retry_wait_max":         cty.NullVal(cty.String),
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	if src, err := ioutil.ReadFile(remoteStatePath); err == nil {
		t.Fatalf("has %s file; should not\n%s", remoteStatePath, src)
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
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
	view, _ := testView(t)
	apply := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               new(cli.MockUi),
			View:             view,
		},
	}
	args := []string{
		planPath,
	}
	apply.Run(args)
}

func TestApply_refresh(t *testing.T) {
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
				AttrsJSON: []byte(`{"ami":"bar"}`),
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
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-shutdown"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	statePath := testTempFile(t)
	p := testProvider()

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
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

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		return
	}

	var once sync.Once
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
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

		resp.NewState = req.PlannedState
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

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	if code := c.Run(args); code != 1 {
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
				AttrsJSON: []byte(`{"ami":"foo"}`),
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
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}
	p.ApplyResourceChangeResponse = &providers.ApplyResourceChangeResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.NullVal(cty.String),
		"ami": cty.StringVal("foo"),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state during plan\ngot: %#v\nwant: %#v", actual, expected)
	}

	actual = p.ApplyResourceChangeRequest.PriorState
	expected = cty.ObjectVal(map[string]cty.Value{
		"id":  cty.NullVal(cty.String),
		"ami": cty.StringVal("foo"),
	})
	if !expected.RawEquals(actual) {
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"idontexist.tfstate",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_sensitiveOutput(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-sensitive-output"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	statePath := testTempFile(t)

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}

	output := done(t).Stdout()
	if !strings.Contains(output, "notsensitive = \"Hello world\"") {
		t.Fatalf("bad: output should contain 'notsensitive' output\n%s", output)
	}
	if !strings.Contains(output, "sensitive = <sensitive>") {
		t.Fatalf("bad: output should contain 'sensitive' output\n%s", output)
	}
}

func TestApply_vars(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	args := []string{
		"-auto-approve",
		"-var", "foo=bar",
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	args := []string{
		"-auto-approve",
		"-var-file", varFilePath,
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefaultJSON(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	varFilePath := filepath.Join(td, "terraform.tfvars.json")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFileJSON), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	actual := ""
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_backup(t *testing.T) {
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
				AttrsJSON: []byte("{\n            \"id\": \"bar\"\n          }"),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, originalState)
	backupPath := testTempFile(t)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", backupPath,
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
	if !cmp.Equal(actual, expected, cmpopts.EquateEmpty()) {
		t.Fatalf(
			"wrong aws_instance.foo state\n%s",
			cmp.Diff(expected, actual, cmp.Transformer("bytesAsString", func(b []byte) string {
				return string(b)
			})),
		)
	}
}

func TestApply_disableBackup(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", "-",
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
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-terraform-env"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
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
	testCopyDir(t, testFixturePath("apply-terraform-env"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Create new env
	{
		ui := new(cli.MockUi)
		view, _ := testView(t)
		newCmd := &WorkspaceNewCommand{}
		newCmd.Meta = Meta{Ui: ui, View: view}
		if code := newCmd.Run([]string{"test"}); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	// Switch to it
	{
		args := []string{"test"}
		ui := new(cli.MockUi)
		view, _ := testView(t)
		selCmd := &WorkspaceSelectCommand{}
		selCmd.Meta = Meta{Ui: ui, View: view}
		if code := selCmd.Run(args); code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter)
		}
	}

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
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

// Config with multiple resources, targeting apply of a subset
func TestApply_targeted(t *testing.T) {
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
	view, _ := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-target", "test_instance.foo",
		"-target", "test_instance.baz",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if got, want := ui.OutputWriter.String(), "3 added, 0 changed, 0 destroyed"; !strings.Contains(got, want) {
		t.Fatalf("bad change summary, want %q, got:\n%s", want, got)
	}
}

// Diagnostics for invalid -target flags
func TestApply_targetFlagsDiags(t *testing.T) {
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
			c := &ApplyCommand{
				Meta: Meta{
					Ui:   ui,
					View: view,
				},
			}

			args := []string{
				"-auto-approve",
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

// applyFixtureSchema returns a schema suitable for processing the
// configuration in testdata/apply . This schema should be
// assigned to a mock provider named "test".
func applyFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
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

// applyFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/apply. This mock has
// GetSchemaResponse, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func applyFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = applyFixtureSchema()
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
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
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
