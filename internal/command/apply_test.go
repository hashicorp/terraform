// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestApply(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
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
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
	if !strings.Contains(output.Stderr(), "-chdir") {
		t.Fatal("expected command output to refer to -chdir flag, but got:", output.Stderr())
	}
}

func TestApply_approveNo(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	defer testInputMap(t, map[string]string{
		"approve": "no",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
	ui := new(cli.MockUi)

	p := applyFixtureProvider()
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
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
	if got, want := output.Stdout(), "Apply cancelled"; !strings.Contains(got, want) {
		t.Fatalf("expected output to include %q, but was:\n%s", want, got)
	}

	if _, err := os.Stat(statePath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("state file should not exist")
	}
}

func TestApply_approveYes(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	defer testInputMap(t, map[string]string{
		"approve": "yes",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	unlock, err := testLockState(t, testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatal("expected error")
	}

	if !strings.Contains(output.Stderr(), "lock") {
		t.Fatal("command output does not look like a lock error:", output.Stderr())
	}
}

// test apply with locked state, waiting for unlock
func TestApply_lockedStateWait(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	unlock, err := testLockState(t, testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}

	// unlock during apply
	go func() {
		time.Sleep(500 * time.Millisecond)
		unlock()
	}()

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("lock should have succeeded in less than 3s: %s", output.Stderr())
	}
}

// Verify that the parallelism flag allows no more than the desired number of
// concurrent calls to ApplyResourceChange.
func TestApply_parallelism(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("parallelism"), td)
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
		provider := &testing_provider.MockProvider{}
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

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: testingOverrides,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		fmt.Sprintf("-parallelism=%d", par),
	}

	res := c.Run(args)
	output := done(t)
	if res != 0 {
		t.Fatal(output.Stdout())
	}
}

func TestApply_configInvalid(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-config-invalid"), td)
	defer testChdir(t, td)()

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", testTempFile(t),
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: \n%s", output.Stdout())
	}
}

func TestApply_defaultState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-error"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("wrong exit code %d; want 1\n%s", code, output.Stdout())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-input"), td)
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-input-partial"), td)
	defer testChdir(t, td)()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// Set some default reader/writers for the inputs
	defaultInputReader = bytes.NewBufferString("one\ntwo\n")
	defaultInputWriter = new(bytes.Buffer)

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-var", "foo=foovalue",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state-out", statePath,
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	statePath := testTempFile(t)
	backupPath := testTempFile(t)

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// create a state file that needs to be backed up
	fs := statemgr.NewFilesystem(statePath)
	fs.StateSnapshotMeta()
	err := fs.WriteState(states.NewState())
	if err != nil {
		t.Fatal(err)
	}

	// the plan file must contain the metadata from the prior state to be
	// backed up
	planPath := applyFixturePlanFileMatchState(t, fs.StateSnapshotMeta())

	args := []string{
		"-state", statePath,
		"-backup", backupPath,
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Should have a backup file
	testStateRead(t, backupPath)
}

func TestApply_plan_noBackup(t *testing.T) {
	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state-out", statePath,
		"-backup", "-",
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	tmp := testCwd(t)
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
		"address":                   cty.StringVal(srv.URL),
		"update_method":             cty.NullVal(cty.String),
		"lock_address":              cty.NullVal(cty.String),
		"unlock_address":            cty.NullVal(cty.String),
		"lock_method":               cty.NullVal(cty.String),
		"unlock_method":             cty.NullVal(cty.String),
		"username":                  cty.NullVal(cty.String),
		"password":                  cty.NullVal(cty.String),
		"skip_cert_verification":    cty.NullVal(cty.Bool),
		"retry_max":                 cty.NullVal(cty.String),
		"retry_wait_min":            cty.NullVal(cty.String),
		"retry_wait_max":            cty.NullVal(cty.String),
		"client_ca_certificate_pem": cty.NullVal(cty.String),
		"client_certificate_pem":    cty.NullVal(cty.String),
		"client_private_key_pem":    cty.NullVal(cty.String),
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state-out", statePath,
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	// This test ensures that it isn't allowed to set input variables
	// when applying from a saved plan file, since in that case the
	// variable values come from the saved plan file.
	//
	// This situation was originally checked by the apply command itself,
	// and that's what this test was originally exercising. This rule
	// is now enforced by the "local" backend instead, but this test
	// is still valid since the command instance delegates to the
	// local backend.

	planPath := applyFixturePlanFile(t)
	statePath := testTempFile(t)

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-var", "foo=bar",
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatal("should've failed: ", output.Stdout())
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
	view, done := testView(t)
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
	done(t)
}

func TestApply_refresh(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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

func TestApply_refreshFalse(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
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
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-refresh=false",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if p.ReadResourceCalled {
		t.Fatal("should not call ReadResource when refresh=false")
	}
}
func TestApply_shutdown(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-shutdown"), td)
	defer testChdir(t, td)()

	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	statePath := testTempFile(t)
	p := testProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
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

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"idontexist.tfstate",
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: \n%s", output.Stdout())
	}
}

func TestApply_sensitiveOutput(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-sensitive-output"), td)
	defer testChdir(t, td)()

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	statePath := testTempFile(t)

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}

	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: \n%s", output.Stdout())
	}

	stdout := output.Stdout()
	if !strings.Contains(stdout, "notsensitive = \"Hello world\"") {
		t.Fatalf("bad: output should contain 'notsensitive' output\n%s", stdout)
	}
	if !strings.Contains(stdout, "sensitive = <sensitive>") {
		t.Fatalf("bad: output should contain 'sensitive' output\n%s", stdout)
	}
}

func TestApply_vars(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer testChdir(t, td)()

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer testChdir(t, td)()

	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_varFileDefaultJSON(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer testChdir(t, td)()

	varFilePath := filepath.Join(td, "terraform.tfvars.json")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFileJSON), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
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
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestApply_backup(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
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
	backupPath := testTempFile(t)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", backupPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()
	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("bar"),
		}),
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-backup", "-",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-terraform-env"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-terraform-env"), td)
	defer testChdir(t, td)()

	// Create new env
	{
		ui := new(cli.MockUi)
		newCmd := &WorkspaceNewCommand{
			Meta: Meta{
				Ui: ui,
			},
		}
		if code := newCmd.Run([]string{"test"}); code != 0 {
			t.Fatal("error creating workspace")
		}
	}

	// Switch to it
	{
		args := []string{"test"}
		ui := new(cli.MockUi)
		selCmd := &WorkspaceSelectCommand{
			Meta: Meta{
				Ui: ui,
			},
		}
		if code := selCmd.Run(args); code != 0 {
			t.Fatal("error switching workspace")
		}
	}

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-targeted"), td)
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

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-target", "test_instance.foo",
		"-target", "test_instance.baz",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if got, want := output.Stdout(), "3 added, 0 changed, 0 destroyed"; !strings.Contains(got, want) {
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

			view, done := testView(t)
			c := &ApplyCommand{
				Meta: Meta{
					View: view,
				},
			}

			args := []string{
				"-auto-approve",
				"-target", target,
			}
			code := c.Run(args)
			output := done(t)
			if code != 1 {
				t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
			}

			got := output.Stderr()
			if !strings.Contains(got, target) {
				t.Fatalf("bad error output, want %q, got:\n%s", target, got)
			}
			if !strings.Contains(got, wantDiag) {
				t.Fatalf("bad error output, want %q, got:\n%s", wantDiag, got)
			}
		})
	}
}

func TestApply_replace(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-replace"), td)
	defer testChdir(t, td)()

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "a",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"hello"}`),
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
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	createCount := 0
	deleteCount := 0
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		if req.PriorState.IsNull() {
			createCount++
		}
		if req.PlannedState.IsNull() {
			deleteCount++
		}
		return providers.ApplyResourceChangeResponse{
			NewState: req.PlannedState,
		}
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
		"-replace", "test_instance.a",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("wrong exit code %d\n\n%s", code, output.Stderr())
	}

	if got, want := output.Stdout(), "1 added, 0 changed, 1 destroyed"; !strings.Contains(got, want) {
		t.Errorf("wrong change summary\ngot output:\n%s\n\nwant substring: %s", got, want)
	}

	if got, want := createCount, 1; got != want {
		t.Errorf("wrong create count %d; want %d", got, want)
	}
	if got, want := deleteCount, 1; got != want {
		t.Errorf("wrong create count %d; want %d", got, want)
	}
}

func TestApply_pluginPath(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	pluginPath := []string{"a", "b", "c"}

	if err := c.Meta.storePluginPath(pluginPath); err != nil {
		t.Fatal(err)
	}
	c.Meta.pluginPath = nil

	args := []string{
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !reflect.DeepEqual(pluginPath, c.Meta.pluginPath) {
		t.Fatalf("expected plugin path %#v, got %#v", pluginPath, c.Meta.pluginPath)
	}
}

func TestApply_jsonGoldenReference(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-json",
		"-state", statePath,
		"-auto-approve",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	checkGoldenReference(t, output, "apply")
}

func TestApply_warnings(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = applyFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.SimpleWarning("warning 1"),
				tfdiags.SimpleWarning("warning 2"),
			},
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: cty.UnknownAsNull(req.PlannedState),
		}
	}

	t.Run("full warnings", func(t *testing.T) {
		view, done := testView(t)
		c := &ApplyCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}

		args := []string{"-auto-approve"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
		wantWarnings := []string{
			"warning 1",
			"warning 2",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(output.Stdout(), want) {
				t.Errorf("missing warning %s", want)
			}
		}
	})

	t.Run("compact warnings", func(t *testing.T) {
		view, done := testView(t)
		c := &ApplyCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}

		code := c.Run([]string{"-auto-approve", "-compact-warnings"})
		output := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
		// the output should contain 2 warnings and a message about -compact-warnings
		wantWarnings := []string{
			"warning 1",
			"warning 2",
			"To see the full warning notes, run Terraform without -compact-warnings.",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(output.Stdout(), want) {
				t.Errorf("missing warning %s", want)
			}
		}
	})
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
func applyFixtureProvider() *testing_provider.MockProvider {
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
	return applyFixturePlanFileMatchState(t, statemgr.SnapshotMeta{})
}

// applyFixturePlanFileMatchState creates a planfile like applyFixturePlanFile,
// but inserts the state meta information if that plan must match a preexisting
// state.
func applyFixturePlanFileMatchState(t *testing.T, stateMeta statemgr.SnapshotMeta) string {
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
	return testPlanFileMatchState(
		t,
		snap,
		states.NewState(),
		plan,
		stateMeta,
	)
}

const applyVarFile = `
foo = "bar"
`

const applyVarFileJSON = `
{ "foo": "bar" }
`
