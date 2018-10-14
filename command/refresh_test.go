package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

func TestRefresh(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(newStateFile.State.String())
	expected := strings.TrimSpace(testRefreshStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestRefresh_empty(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("refresh-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		td,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestRefresh_lockedState(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	unlock, err := testLockState("./testdata", statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		testFixturePath("refresh"),
	}

	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "lock") {
		t.Fatal("command output does not look like a lock error:", output)
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newStateFile, err := statefile.Read(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(newStateFile.State.String())
	expected := strings.TrimSpace(testRefreshCwdStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestRefresh_defaultState(t *testing.T) {
	originalState := testState()

	// Write the state file in a temporary directory with the
	// default filename.
	statePath := testStateFile(t, originalState)

	localState := statemgr.NewFilesystem(statePath)
	if err := localState.RefreshState(); err != nil {
		t.Fatal(err)
	}
	s := localState.State()
	if s == nil {
		t.Fatal("empty test state")
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}

	newState := testStateRead(t, statePath)

	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte("{\n            \"ami\": null,\n            \"id\": \"yes\"\n          }"),
		Dependencies: []addrs.Referenceable{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	backupState := testStateRead(t, statePath+DefaultBackupExtension)

	actual = backupState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected = originalState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}
}

func TestRefresh_outPath(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile(testingDir, "tf")
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	newState := testStateRead(t, statePath)
	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte("{\n            \"ami\": null,\n            \"id\": \"yes\"\n          }"),
		Dependencies: []addrs.Referenceable{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	backupState := testStateRead(t, outPath + DefaultBackupExtension)
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}
	p.GetSchemaReturn = refreshVarFixtureSchema()

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
	if got, want := p.ConfigureRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRefresh_varFile(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}
	p.GetSchemaReturn = refreshVarFixtureSchema()

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
	if got, want := p.ConfigureRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRefresh_varFileDefault(t *testing.T) {
	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}
	p.GetSchemaReturn = refreshVarFixtureSchema()

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
	if got, want := p.ConfigureRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
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
	outf, err := ioutil.TempFile(testingDir, "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	// Backup path
	backupf, err := ioutil.TempFile(testingDir, "tf")
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("changed"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", backupPath,
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	newState := testStateRead(t, statePath)
	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte("{\n            \"ami\": null,\n            \"id\": \"changed\"\n          }"),
		Dependencies: []addrs.Referenceable{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	backupState := testStateRead(t, outPath + DefaultBackupExtension)
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
	outf, err := ioutil.TempFile(testingDir, "tf")
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.GetSchemaReturn = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", "-",
		testFixturePath("refresh"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	newState := testStateRead(t, statePath)
	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte("{\n            \"ami\": null,\n            \"id\": \"yes\"\n          }"),
		Dependencies: []addrs.Referenceable{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	// Ensure there is no backup
	_, err = os.Stat(outPath + DefaultBackupExtension)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("backup should not exist")
	}
	_, err = os.Stat("-")
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
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
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

// newInstanceState creates a new states.ResourceInstanceObjectSrc with the
// given value for its single id attribute. It is named newInstanceState for
// historical reasons, because it was originally written for the poorly-named
// terraform.InstanceState type.
func newInstanceState(id string) *states.ResourceInstanceObjectSrc {
	attrs := map[string]interface{}{
		"id": id,
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal attributes: %s", err)) // should never happen
	}
	return &states.ResourceInstanceObjectSrc{
		AttrsJSON: attrsJSON,
		Status:    states.ObjectReady,
	}
}

// refreshFixtureSchema returns a schema suitable for processing the
// configuration in test-fixtures/refresh . This schema should be
// assigned to a mock provider named "test".
func refreshFixtureSchema() *terraform.ProviderSchema {
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

// refreshVarFixtureSchema returns a schema suitable for processing the
// configuration in test-fixtures/refresh-var . This schema should be
// assigned to a mock provider named "test".
func refreshVarFixtureSchema() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Optional: true, Computed: true},
				},
			},
		},
	}
}

const refreshVarFile = `
foo = "bar"
`

const testRefreshStr = `
test_instance.foo:
  ID = yes
  provider = provider.test
`
const testRefreshCwdStr = `
test_instance.foo:
  ID = yes
  provider = provider.test
`
