package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestRefresh(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-state", statePath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRefresh_badState(t *testing.T) {
	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-state", "i-should-not-exist-ever",
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestRefresh_cwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("refresh")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRefresh_defaultState(t *testing.T) {
	originalState := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}

	// Write the state file in a temporary directory with the
	// default filename.
	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	statePath := filepath.Join(td, DefaultStateFilename)

	f, err := os.Create(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = terraform.WriteState(originalState, f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	f, err = os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	f, err = os.Open(statePath + DefaultBackupExtention)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual = backupState.Resources["test_instance.foo"]
	expected = originalState.Resources["test_instance.foo"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRefresh_outPath(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	f, err = os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err = terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	f, err = os.Open(outPath + DefaultBackupExtention)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(backupState, state) {
		t.Fatalf("bad: %#v", backupState)
	}
}

func TestRefresh_var(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	args := []string{
		"-var", "foo=bar",
		"-state", statePath,
		testFixturePath("refresh-var"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if p.ConfigureConfig.Config["value"].(string) != "bar" {
		t.Fatalf("bad: %#v", p.ConfigureConfig.Config)
	}
}

func TestRefresh_varFile(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(refreshVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	args := []string{
		"-var-file", varFilePath,
		"-state", statePath,
		testFixturePath("refresh-var"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if p.ConfigureConfig.Config["value"].(string) != "bar" {
		t.Fatalf("bad: %#v", p.ConfigureConfig.Config)
	}
}

func TestRefresh_varFileDefault(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	varFileDir := testTempDir(t)
	varFilePath := filepath.Join(varFileDir, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(refreshVarFile), 0644); err != nil {
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

	args := []string{
		"-state", statePath,
		testFixturePath("refresh-var"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if p.ConfigureConfig.Config["value"].(string) != "bar" {
		t.Fatalf("bad: %#v", p.ConfigureConfig.Config)
	}
}

func TestRefresh_backup(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	// Backup path
	backupf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	backupPath := backupf.Name()
	backupf.Close()
	os.Remove(backupPath)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", backupPath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	f, err = os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err = terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	f, err = os.Open(backupPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(backupState, state) {
		t.Fatalf("bad: %#v", backupState)
	}
}

func TestRefresh_disableBackup(t *testing.T) {
	state := &terraform.State{
		Resources: map[string]*terraform.ResourceState{
			"test_instance.foo": &terraform.ResourceState{
				ID:   "bar",
				Type: "test_instance",
			},
		},
	}
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.ResourceState{ID: "yes"}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", "-",
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	f, err = os.Open(outPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err = terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := newState.Resources["test_instance.foo"]
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	// Ensure there is no backup
	_, err = os.Stat(outPath + DefaultBackupExtention)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
}

const refreshVarFile = `
foo = "bar"
`
