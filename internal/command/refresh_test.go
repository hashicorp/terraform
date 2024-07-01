// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var equateEmpty = cmpopts.EquateEmpty()

func TestRefresh(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-empty"), td)
	defer testChdir(t, td)()

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestRefresh_lockedState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	unlock, err := testLockState(t, testDataDir, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
	}

	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatal("expected error")
	}

	got := output.Stderr()
	if !strings.Contains(got, "lock") {
		t.Fatal("command output does not look like a lock error:", got)
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
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
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
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

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
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}

	newState := testStateRead(t, statePath)

	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte(`{"ami":null,"id":"yes"}`),
		Dependencies: []addrs.ConfigResource{},
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
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile(td, "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	newState := testStateRead(t, statePath)
	if !reflect.DeepEqual(newState, state) {
		t.Fatalf("bad: %#v", newState)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte(`{"ami":null,"id":"yes"}`),
		Dependencies: []addrs.ConfigResource{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	if _, err := os.Stat(outPath + DefaultBackupExtension); !os.IsNotExist(err) {
		if err != nil {
			t.Fatalf("failed to test for backup file: %s", err)
		}
		t.Fatalf("backup file exists, but it should not because output file did not initially exist")
	}
}

func TestRefresh_var(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-var"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}
	p.GetProviderSchemaResponse = refreshVarFixtureSchema()

	args := []string{
		"-var", "foo=bar",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("configure should be called")
	}
	if got, want := p.ConfigureProviderRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRefresh_varFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-var"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}
	p.GetProviderSchemaResponse = refreshVarFixtureSchema()

	varFilePath := testTempFile(t)
	if err := ioutil.WriteFile(varFilePath, []byte(refreshVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	args := []string{
		"-var-file", varFilePath,
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("configure should be called")
	}
	if got, want := p.ConfigureProviderRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRefresh_varFileDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-var"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}
	p.GetProviderSchemaResponse = refreshVarFixtureSchema()

	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(refreshVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("configure should be called")
	}
	if got, want := p.ConfigureProviderRequest.Config.GetAttr("value"), cty.StringVal("bar"); !want.RawEquals(got) {
		t.Fatalf("wrong provider configuration\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRefresh_varsUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-unset-var"), td)
	defer testChdir(t, td)()

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	defaultInputReader = bytes.NewBufferString("bar\n")

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
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

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}

func TestRefresh_backup(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile(td, "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	defer outf.Close()

	// Need to put some state content in the output file so that there's
	// something to back up.
	err = statefile.Write(statefile.New(state, "baz", 0), outf)
	if err != nil {
		t.Fatalf("error writing initial output state file %s", err)
	}

	// Backup path
	backupf, err := ioutil.TempFile(td, "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	backupPath := backupf.Name()
	backupf.Close()
	os.Remove(backupPath)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("changed"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", backupPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	newState := testStateRead(t, statePath)
	if !cmp.Equal(newState, state, cmpopts.EquateEmpty()) {
		t.Fatalf("got:\n%s\nexpected:\n%s\n", newState, state)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte(`{"ami":null,"id":"changed"}`),
		Dependencies: []addrs.ConfigResource{},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong new object\ngot:  %swant: %s", spew.Sdump(actual), spew.Sdump(expected))
	}

	backupState := testStateRead(t, backupPath)
	actualStr := strings.TrimSpace(backupState.String())
	expectedStr := strings.TrimSpace(state.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestRefresh_disableBackup(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	// Output path
	outf, err := ioutil.TempFile(td, "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	outPath := outf.Name()
	outf.Close()
	os.Remove(outPath)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("yes"),
		}),
	}

	args := []string{
		"-state", statePath,
		"-state-out", outPath,
		"-backup", "-",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	newState := testStateRead(t, statePath)
	if !cmp.Equal(state, newState, equateEmpty) {
		spew.Config.DisableMethods = true
		fmt.Println(cmp.Diff(state, newState, equateEmpty))
		t.Fatalf("bad: %s", newState)
	}

	newState = testStateRead(t, outPath)
	actual := newState.RootModule().Resources["test_instance.foo"].Instances[addrs.NoKey].Current
	expected := &states.ResourceInstanceObjectSrc{
		Status:       states.ObjectReady,
		AttrsJSON:    []byte(`{"ami":null,"id":"yes"}`),
		Dependencies: []addrs.ConfigResource{},
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
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-output"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

	p := testProvider()
	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
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

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Test that outputs were displayed
	outputValue := "foo.example.com"
	actual := output.Stdout()
	if !strings.Contains(actual, outputValue) {
		t.Fatalf("Expected:\n%s\n\nTo include: %q", actual, outputValue)
	}
}

// Config with multiple resources, targeting refresh of a subset
func TestRefresh_targeted(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("refresh-targeted"), td)
	defer testChdir(t, td)()

	state := testState()
	statePath := testStateFile(t, state)

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
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-target", "test_instance.foo",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	got := output.Stdout()
	if want := "test_instance.foo: Refreshing"; !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, got)
	}
	if doNotWant := "test_instance.bar: Refreshing"; strings.Contains(got, doNotWant) {
		t.Fatalf("expected output not to contain %q, got:\n%s", doNotWant, got)
	}
}

// Diagnostics for invalid -target flags
func TestRefresh_targetFlagsDiags(t *testing.T) {
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
			c := &RefreshCommand{
				Meta: Meta{
					View: view,
				},
			}

			args := []string{
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

func TestRefresh_warnings(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = refreshFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.SimpleWarning("warning 1"),
				tfdiags.SimpleWarning("warning 2"),
			},
		}
	}

	t.Run("full warnings", func(t *testing.T) {
		view, done := testView(t)
		c := &RefreshCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}

		code := c.Run([]string{})
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
		c := &RefreshCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}

		code := c.Run([]string{"-compact-warnings"})
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

// configuration in testdata/refresh . This schema should be
// assigned to a mock provider named "test".
func refreshFixtureSchema() *providers.GetProviderSchemaResponse {
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

// refreshVarFixtureSchema returns a schema suitable for processing the
// configuration in testdata/refresh-var . This schema should be
// assigned to a mock provider named "test".
func refreshVarFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
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
}

const refreshVarFile = `
foo = "bar"
`

const testRefreshStr = `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
`
const testRefreshCwdStr = `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
`
