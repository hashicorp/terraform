package command

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestImport(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigDisable(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); ok {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"-state", statePath,
		"-config", "",
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigWithVar(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"-state", statePath,
		"-var", "foo=bar",
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigWithVarDefault(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var-default"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigWithVarFile(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var-file"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"-state", statePath,
		"-var-file", "blah.tfvars",
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

/*
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

	state := testState()
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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := strings.TrimSpace(newState.String())
	expected := strings.TrimSpace(testRefreshCwdStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestRefresh_defaultState(t *testing.T) {
	originalState := testState()

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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := newState.RootModule().Resources["test_instance.foo"].Primary
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual = backupState.RootModule().Resources["test_instance.foo"].Primary
	expected = originalState.RootModule().Resources["test_instance.foo"].Primary
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRefresh_futureState(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("refresh")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	state := testState()
	state.TFVersion = "99.99.99"
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
		"-state", statePath,
	}
	if code := c.Run(args); code == 0 {
		t.Fatal("should fail")
	}

	if p.RefreshCalled {
		t.Fatal("refresh should not be called")
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

	actual := strings.TrimSpace(newState.String())
	expected := strings.TrimSpace(state.String())
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestRefresh_pastState(t *testing.T) {
	state := testState()
	state.TFVersion = "0.1.0"
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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := strings.TrimSpace(newState.String())
	expected := strings.TrimSpace(testRefreshStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}

	if newState.TFVersion != terraform.Version {
		t.Fatalf("bad:\n\n%s", newState.TFVersion)
	}
}

func TestRefresh_outPath(t *testing.T) {
	state := testState()
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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := newState.RootModule().Resources["test_instance.foo"].Primary
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	f, err = os.Open(outPath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr := strings.TrimSpace(backupState.String())
	expectedStr := strings.TrimSpace(state.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestRefresh_var(t *testing.T) {
	state := testState()
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
	state := testState()
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
	state := testState()
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

func TestRefresh_varsUnset(t *testing.T) {
	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	defaultInputReader = bytes.NewBufferString("bar\n")

	state := testState()
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
		"-state", statePath,
		testFixturePath("refresh-unset-var"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestRefresh_backup(t *testing.T) {
	state := testState()
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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := newState.RootModule().Resources["test_instance.foo"].Primary
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

	actualStr := strings.TrimSpace(backupState.String())
	expectedStr := strings.TrimSpace(state.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestRefresh_disableBackup(t *testing.T) {
	state := testState()
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
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

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

	actual := newState.RootModule().Resources["test_instance.foo"].Primary
	expected := p.RefreshReturn
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	// Ensure there is no backup
	_, err = os.Stat(outPath + DefaultBackupExtension)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
}

func TestRefresh_displaysOutputs(t *testing.T) {
	state := testState()
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
		"-state", statePath,
		testFixturePath("refresh-output"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that outputs were displayed
	outputValue := "foo.example.com"
	actual := ui.OutputWriter.String()
	if !strings.Contains(actual, outputValue) {
		t.Fatalf("Expected:\n%s\n\nTo include: %q", actual, outputValue)
	}
}
*/

func TestImport_customProvider(t *testing.T) {
	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
		},
	}

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
			},
		},
	}

	args := []string{
		"-provider", "test.alias",
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportCustomProviderStr)
}

const testImportStr = `
test_instance.foo:
  ID = yay
  provider = test
`

const testImportCustomProviderStr = `
test_instance.foo:
  ID = yay
  provider = test.alias
`
