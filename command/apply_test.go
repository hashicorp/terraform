package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestApply(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-state", statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_configInvalid(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-state", testTempFile(t),
		testFixturePath("apply-config-invalid"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestApply_defaultState(t *testing.T) {
	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
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

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_error(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	var lock sync.Mutex
	errored := false
	p.ApplyFn = func(
		s *terraform.ResourceState,
		d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
		lock.Lock()
		defer lock.Unlock()

		if !errored {
			errored = true
			return nil, fmt.Errorf("error")
		}

		return &terraform.ResourceState{ID: "foo"}, nil
	}
	p.DiffFn = func(
		*terraform.ResourceState,
		*terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		return &terraform.ResourceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	args := []string{
		"-state", statePath,
		testFixturePath("apply-error"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
	if len(state.Resources) == 0 {
		t.Fatal("no resources in state")
	}
}

func TestApply_noArgs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("plan")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_plan(t *testing.T) {
	planPath := testPlanFile(t, &terraform.Plan{
		Config: new(config.Config),
	})
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-state", statePath,
		planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestApply_planVars(t *testing.T) {
	planPath := testPlanFile(t, &terraform.Plan{
		Config: new(config.Config),
	})
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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

func TestApply_refresh(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-state", statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.RefreshCalled {
		t.Fatal("should call refresh")
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtention)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(backupState, originalState) {
		t.Fatalf("bad: %#v", backupState)
	}
}

func TestApply_shutdown(t *testing.T) {
	stopped := false
	stopCh := make(chan struct{})
	stopReplyCh := make(chan struct{})

	statePath := testTempFile(t)

	p := testProvider()
	shutdownCh := make(chan struct{})
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},

		ShutdownCh: shutdownCh,
	}

	p.DiffFn = func(
		*terraform.ResourceState,
		*terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		return &terraform.ResourceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}
	p.ApplyFn = func(
		*terraform.ResourceState,
		*terraform.ResourceDiff) (*terraform.ResourceState, error) {
		if !stopped {
			stopped = true
			close(stopCh)
			<-stopReplyCh
		}

		return &terraform.ResourceState{
			ID: "foo",
			Attributes: map[string]string{
				"ami": "2",
			},
		}, nil
	}

	go func() {
		<-stopCh
		shutdownCh <- struct{}{}

		// This is really dirty, but we have no other way to assure that
		// tf.Stop() has been called. This doesn't assure it either, but
		// it makes it much more certain.
		time.Sleep(50 * time.Millisecond)

		close(stopReplyCh)
	}()

	args := []string{
		"-state", statePath,
		testFixturePath("apply-shutdown"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	if len(state.Resources) != 1 {
		t.Fatalf("bad: %d", len(state.Resources))
	}
}

func TestApply_state(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:       "bar",
				Type:     "test_instance",
				ConnInfo: make(map[string]string),
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.DiffReturn = &terraform.ResourceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"ami": &terraform.ResourceAttrDiff{
				New: "bar",
			},
		},
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	expectedState := originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(p.DiffState, expectedState) {
		t.Fatalf("bad: %#v", p.DiffState)
	}

	if !reflect.DeepEqual(p.ApplyState, expectedState) {
		t.Fatalf("bad: %#v", p.ApplyState)
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

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtention)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// nil out the ConnInfo since that should not be restored
	originalState.Resources["test_instance.foo"].ConnInfo = nil

	if !reflect.DeepEqual(backupState, originalState) {
		t.Fatalf("bad: %#v", backupState)
	}
}

func TestApply_stateNoExist(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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

func TestApply_vars(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		s *terraform.ResourceState,
		c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.ResourceDiff{}, nil
	}

	args := []string{
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
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		s *terraform.ResourceState,
		c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.ResourceDiff{}, nil
	}

	args := []string{
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
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	actual := ""
	p.DiffFn = func(
		s *terraform.ResourceState,
		c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
		if v, ok := c.Config["value"]; ok {
			actual = v.(string)
		}

		return &terraform.ResourceDiff{}, nil
	}

	args := []string{
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
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	statePath := testStateFile(t, originalState)
	backupPath := testTempFile(t)

	p := testProvider()
	p.DiffReturn = &terraform.ResourceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"ami": &terraform.ResourceAttrDiff{
				New: "bar",
			},
		},
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
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

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Should have a backup file
	f, err = os.Open(backupPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := backupState.Resources["test_instance.foo"]
	expected := originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v %#v", actual, expected)
	}
}

func TestApply_disableBackup(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:       "bar",
				Type:     "test_instance",
				ConnInfo: make(map[string]string),
			},
		},
	}

	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.DiffReturn = &terraform.ResourceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"ami": &terraform.ResourceAttrDiff{
				New: "bar",
			},
		},
	}

	ui := new(cli.MockUi)
	c := &ApplyCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-state", statePath,
		"-backup", "-",
		testFixturePath("apply"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that the provider was called with the existing state
	expectedState := originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(p.DiffState, expectedState) {
		t.Fatalf("bad: %#v", p.DiffState)
	}

	if !reflect.DeepEqual(p.ApplyState, expectedState) {
		t.Fatalf("bad: %#v", p.ApplyState)
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

	state, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// Ensure there is no backup
	_, err = os.Stat(statePath + DefaultBackupExtention)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
}

const applyVarFile = `
foo = "bar"
`
