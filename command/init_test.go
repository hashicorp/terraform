package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

func TestInit_empty(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestInit_multipleArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestInit_fromModule_explicitDest(t *testing.T) {
	dir := tempDir(t)
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if _, err := os.Stat(DefaultStateFilename); err == nil {
		// This should never happen; it indicates a bug in another test
		// is causing a terraform.tfstate to get left behind in our directory
		// here, which can interfere with our init process in a way that
		// isn't relevant to this test.
		t.Fatalf("some other test has left terraform.tfstate behind")
	}

	args := []string{
		"-from-module=" + testFixturePath("init"),
		dir,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(dir, "hello.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestInit_fromModule_cwdDest(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, os.ModePerm)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-from-module=" + testFixturePath("init"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(td, "hello.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// https://github.com/hashicorp/terraform/issues/518
func TestInit_fromModule_dstInSrc(t *testing.T) {
	dir := tempDir(t)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Change to the temporary directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	if err := os.Mkdir("foo", os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Create("issue518.tf"); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-from-module=.",
		"foo",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(dir, "foo", "issue518.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestInit_get(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Check output
	output := ui.OutputWriter.String()
	if !strings.Contains(output, "foo in foo") {
		t.Fatalf("doesn't look like we installed module 'foo': %s", output)
	}
}

func TestInit_getUpgradeModules(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	// copy.CopyDir(testFixturePath("init-get"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-get=true",
		"-get-plugins=false",
		"-upgrade",
		testFixturePath("init-get"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("command did not complete successfully:\n%s", ui.ErrorWriter.String())
	}

	// Check output
	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Upgrading modules...") {
		t.Fatalf("doesn't look like get upgrade: %s", output)
	}
}

func TestInit_backend(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestInit_backendUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	{
		log.Printf("[TRACE] TestInit_backendUnset: beginning first init")

		ui := cli.NewMockUi()
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
			},
		}

		// Init
		args := []string{}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
		}
		log.Printf("[TRACE] TestInit_backendUnset: first init complete")
		t.Logf("First run output:\n%s", ui.OutputWriter.String())
		t.Logf("First run errors:\n%s", ui.ErrorWriter.String())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		log.Printf("[TRACE] TestInit_backendUnset: beginning second init")

		// Unset
		if err := ioutil.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
			},
		}

		args := []string{"-force-copy"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
		}
		log.Printf("[TRACE] TestInit_backendUnset: second init complete")
		t.Logf("Second run output:\n%s", ui.OutputWriter.String())
		t.Logf("Second run errors:\n%s", ui.ErrorWriter.String())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if !s.Backend.Empty() {
			t.Fatal("should not have backend config")
		}
	}
}

func TestInit_backendConfigFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-config-file"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-backend-config", "input.config"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestInit_backendConfigFileChange(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-config-file-change"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInputMap(t, map[string]string{
		"backend-migrate-to-new": "no",
	})()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-backend-config", "input.config"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestInit_backendConfigKV(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-config-kv"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-backend-config", "path=hello"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestInit_targetSubdir(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// copy the source into a subdir
	copy.CopyDir(testFixturePath("init-backend"), filepath.Join(td, "source"))

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"source",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if _, err := os.Stat(filepath.Join(td, DefaultDataDir, DefaultStateFilename)); err != nil {
		t.Fatalf("err: %s", err)
	}

	// a data directory should not have been added to out working dir
	if _, err := os.Stat(filepath.Join(td, "source", DefaultDataDir)); !os.IsNotExist(err) {
		t.Fatalf("err: %s", err)
	}
}

func TestInit_backendReinitWithExtra(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend-empty"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	m := testMetaBackend(t, nil)
	opts := &BackendOpts{
		ConfigOverride: configs.SynthBody("synth", map[string]cty.Value{
			"path": cty.StringVal("hello"),
		}),
		Init: true,
	}

	_, cHash, err := m.backendConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-backend-config", "path=hello"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}

	if state.Backend.Hash != uint64(cHash) {
		t.Fatal("mismatched state and config backend hashes")
	}

	// init again and make sure nothing changes
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
	state = testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
	if state.Backend.Hash != uint64(cHash) {
		t.Fatal("mismatched state and config backend hashes")
	}
}

// move option from config to -backend-config args
func TestInit_backendReinitConfigToExtra(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{"-input=false"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"foo","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}

	backendHash := state.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := ioutil.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// We need a fresh InitCommand here because the old one now has our configuration
	// file cached inside it, so it won't re-read the modification we just made.
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-input=false", "-backend-config=path=foo"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
	state = testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"foo","workspace_dir":null}`; got != want {
		t.Errorf("wrong config after moving to arg\ngot:  %s\nwant: %s", got, want)
	}

	if state.Backend.Hash == backendHash {
		t.Fatal("state.Backend.Hash was not updated")
	}
}

// make sure inputFalse stops execution on migrate
func TestInit_inputFalse(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-backend"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{"-input=false", "-backend-config=path=foo"}
	if code := c.Run([]string{"-input=false"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter)
	}

	// write different states for foo and bar
	fooState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("foo"),
			false, // not sensitive
		)
	})
	if err := statemgr.NewFilesystem("foo").WriteState(fooState); err != nil {
		t.Fatal(err)
	}
	barState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false, // not sensitive
		)
	})
	if err := statemgr.NewFilesystem("bar").WriteState(barState); err != nil {
		t.Fatal(err)
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args = []string{"-input=false", "-backend-config=path=bar"}
	if code := c.Run(args); code == 0 {
		t.Fatal("init should have failed", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "input disabled") {
		t.Fatal("expected input disabled error, got", errMsg)
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	// A missing input=false should abort rather than loop infinitely
	args = []string{"-backend-config=path=bar"}
	if code := c.Run(args); code == 0 {
		t.Fatal("init should have failed", ui.OutputWriter)
	}
}

func TestInit_getProvider(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	overrides := metaOverridesForProvider(testProvider())
	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: overrides,
		Ui:               ui,
	}
	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			// looking for an exact version
			"exact": []string{"1.2.3"},
			// config requires >= 2.3.3
			"greater_than": []string{"2.3.4", "2.3.3", "2.3.0"},
			// config specifies
			"between": []string{"3.4.5", "2.3.4", "1.2.3"},
		},

		Dir: m.pluginDir(),
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{
		"-backend=false", // should be possible to install plugins without backend init
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	if !installer.PurgeUnusedCalled {
		t.Errorf("init didn't purge providers, but should have")
	}

	// check that we got the providers for our config
	exactPath := filepath.Join(c.pluginDir(), installer.FileName("exact", "1.2.3"))
	if _, err := os.Stat(exactPath); os.IsNotExist(err) {
		t.Fatal("provider 'exact' not downloaded")
	}
	greaterThanPath := filepath.Join(c.pluginDir(), installer.FileName("greater_than", "2.3.4"))
	if _, err := os.Stat(greaterThanPath); os.IsNotExist(err) {
		t.Fatal("provider 'greater_than' not downloaded")
	}
	betweenPath := filepath.Join(c.pluginDir(), installer.FileName("between", "2.3.4"))
	if _, err := os.Stat(betweenPath); os.IsNotExist(err) {
		t.Fatal("provider 'between' not downloaded")
	}

	t.Run("future-state", func(t *testing.T) {
		// getting providers should fail if a state from a newer version of
		// terraform exists, since InitCommand.getProviders needs to inspect that
		// state.
		s := terraform.NewState()
		s.TFVersion = "100.1.0"
		local := &state.LocalState{
			Path: local.DefaultStateFilename,
		}
		if err := local.WriteState(s); err != nil {
			t.Fatal(err)
		}

		ui := new(cli.MockUi)
		m.Ui = ui
		c := &InitCommand{
			Meta:              m,
			providerInstaller: installer,
		}

		if code := c.Run(nil); code == 0 {
			t.Fatal("expected error, got:", ui.OutputWriter)
		}

		errMsg := ui.ErrorWriter.String()
		if !strings.Contains(errMsg, "which is newer than current") {
			t.Fatal("unexpected error:", errMsg)
		}
	})
}

// make sure we can locate providers in various paths
func TestInit_findVendoredProviders(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)

	configDirName := "init-get-providers"
	copy.CopyDir(testFixturePath(configDirName), filepath.Join(td, configDirName))
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: &mockProviderInstaller{},
	}

	// make our plugin paths
	if err := os.MkdirAll(c.pluginDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(DefaultPluginVendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	// add some dummy providers
	// the auto plugin directory
	exactPath := filepath.Join(c.pluginDir(), "terraform-provider-exact_v1.2.3_x4")
	if err := ioutil.WriteFile(exactPath, []byte("test bin"), 0755); err != nil {
		t.Fatal(err)
	}
	// the vendor path
	greaterThanPath := filepath.Join(DefaultPluginVendorDir, "terraform-provider-greater_than_v2.3.4_x4")
	if err := ioutil.WriteFile(greaterThanPath, []byte("test bin"), 0755); err != nil {
		t.Fatal(err)
	}
	// Check the current directory too
	betweenPath := filepath.Join(".", "terraform-provider-between_v2.3.4_x4")
	if err := ioutil.WriteFile(betweenPath, []byte("test bin"), 0755); err != nil {
		t.Fatal(err)
	}

	args := []string{configDirName}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

// make sure we can locate providers defined in the legacy rc file
func TestInit_rcProviders(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)

	configDirName := "init-legacy-rc"
	copy.CopyDir(testFixturePath(configDirName), filepath.Join(td, configDirName))
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	pluginDir := filepath.Join(td, "custom")
	pluginPath := filepath.Join(pluginDir, "terraform-provider-legacy")

	ui := new(cli.MockUi)
	m := Meta{
		Ui: ui,
		PluginOverrides: &PluginOverrides{
			Providers: map[string]string{
				"legacy": pluginPath,
			},
		},
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: &mockProviderInstaller{},
	}

	// make our plugin paths
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(pluginPath, []byte("test bin"), 0755); err != nil {
		t.Fatal(err)
	}

	args := []string{configDirName}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestInit_getUpgradePlugins(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			// looking for an exact version
			"exact": []string{"1.2.3"},
			// config requires >= 2.3.3
			"greater_than": []string{"2.3.4", "2.3.3", "2.3.0"},
			// config specifies
			"between": []string{"3.4.5", "2.3.4", "1.2.3"},
		},

		Dir: m.pluginDir(),
	}

	err := os.MkdirAll(m.pluginDir(), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	exactUnwanted := filepath.Join(m.pluginDir(), installer.FileName("exact", "0.0.1"))
	err = ioutil.WriteFile(exactUnwanted, []byte{}, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	greaterThanUnwanted := filepath.Join(m.pluginDir(), installer.FileName("greater_than", "2.3.3"))
	err = ioutil.WriteFile(greaterThanUnwanted, []byte{}, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	betweenOverride := installer.FileName("between", "2.3.4") // intentionally directly in cwd, and should override auto-install
	err = ioutil.WriteFile(betweenOverride, []byte{}, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{
		"-upgrade=true",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("command did not complete successfully:\n%s", ui.ErrorWriter.String())
	}

	files, err := ioutil.ReadDir(m.pluginDir())
	if err != nil {
		t.Fatal(err)
	}

	if !installer.PurgeUnusedCalled {
		t.Errorf("init -upgrade didn't purge providers, but should have")
	}

	gotFilenames := make([]string, len(files))
	for i, info := range files {
		gotFilenames[i] = info.Name()
	}
	sort.Strings(gotFilenames)

	wantFilenames := []string{
		"lock.json",

		// no "between" because the file in cwd overrides it

		// The mock PurgeUnused doesn't actually purge anything, so the dir
		// includes both our old and new versions.
		"terraform-provider-exact_v0.0.1_x4",
		"terraform-provider-exact_v1.2.3_x4",
		"terraform-provider-greater_than_v2.3.3_x4",
		"terraform-provider-greater_than_v2.3.4_x4",
	}

	if !reflect.DeepEqual(gotFilenames, wantFilenames) {
		t.Errorf("wrong directory contents after upgrade\ngot:  %#v\nwant: %#v", gotFilenames, wantFilenames)
	}

}

func TestInit_getProviderMissing(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			// looking for exact version 1.2.3
			"exact": []string{"1.2.4"},
			// config requires >= 2.3.3
			"greater_than": []string{"2.3.4", "2.3.3", "2.3.0"},
			// config specifies
			"between": []string{"3.4.5", "2.3.4", "1.2.3"},
		},

		Dir: m.pluginDir(),
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expceted error, got output: \n%s", ui.OutputWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "no suitable version for provider") {
		t.Fatalf("unexpected error output: %s", ui.ErrorWriter)
	}
}

func TestInit_getProviderHaveLegacyVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-providers-lock"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	if err := ioutil.WriteFile("terraform-provider-test", []byte("provider bin"), 0755); err != nil {
		t.Fatal(err)
	}

	// provider test has a version constraint in the config, which should
	// trigger the getProvider error below.
	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
		providerInstaller: callbackPluginInstaller(func(provider string, req discovery.Constraints) (discovery.PluginMeta, error) {
			return discovery.PluginMeta{}, fmt.Errorf("EXPECTED PROVIDER ERROR %s", provider)
		}),
	}

	args := []string{}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expceted error, got output: \n%s", ui.OutputWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "EXPECTED PROVIDER ERROR test") {
		t.Fatalf("unexpected error output: %s", ui.ErrorWriter)
	}
}

func TestInit_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-check-required-version"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := cli.NewMockUi()
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, ui.ErrorWriter.String(), ui.OutputWriter.String())
	}
}

func TestInit_providerLockFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-provider-lock-file"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			"test": []string{"1.2.3"},
		},

		Dir: m.pluginDir(),
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	providersLockFile := fmt.Sprintf(
		".terraform/plugins/%s_%s/lock.json",
		runtime.GOOS, runtime.GOARCH,
	)
	buf, err := ioutil.ReadFile(providersLockFile)
	if err != nil {
		t.Fatalf("failed to read providers lock file %s: %s", providersLockFile, err)
	}
	// The hash in here is for the empty files that mockGetProvider produces
	wantLockFile := strings.TrimSpace(`
{
  "test": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
`)
	if string(buf) != wantLockFile {
		t.Errorf("wrong provider lock file contents\ngot:  %s\nwant: %s", buf, wantLockFile)
	}
}

func TestInit_pluginDirReset(t *testing.T) {
	td := testTempDir(t)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
		providerInstaller: &mockProviderInstaller{},
	}

	// make our vendor paths
	pluginPath := []string{"a", "b", "c"}
	for _, p := range pluginPath {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// run once and save the -plugin-dir
	args := []string{"-plugin-dir", "a"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter)
	}

	pluginDirs, err := c.loadPluginPath()
	if err != nil {
		t.Fatal(err)
	}

	if len(pluginDirs) != 1 || pluginDirs[0] != "a" {
		t.Fatalf(`expected plugin dir ["a"], got %q`, pluginDirs)
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
		providerInstaller: &mockProviderInstaller{},
	}

	// make sure we remove the plugin-dir record
	args = []string{"-plugin-dir="}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter)
	}

	pluginDirs, err = c.loadPluginPath()
	if err != nil {
		t.Fatal(err)
	}

	if len(pluginDirs) != 0 {
		t.Fatalf("expected no plugin dirs got %q", pluginDirs)
	}
}

// Test user-supplied -plugin-dir
func TestInit_pluginDirProviders(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: &mockProviderInstaller{},
	}

	// make our vendor paths
	pluginPath := []string{"a", "b", "c"}
	for _, p := range pluginPath {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// add some dummy providers in our plugin dirs
	for i, name := range []string{
		"terraform-provider-exact_v1.2.3_x4",
		"terraform-provider-greater_than_v2.3.4_x4",
		"terraform-provider-between_v2.3.4_x4",
	} {

		if err := ioutil.WriteFile(filepath.Join(pluginPath[i], name), []byte("test bin"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{
		"-plugin-dir", "a",
		"-plugin-dir", "b",
		"-plugin-dir", "c",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter)
	}
}

// Test user-supplied -plugin-dir doesn't allow auto-install
func TestInit_pluginDirProvidersDoesNotGet(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-get-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	c := &InitCommand{
		Meta: m,
		providerInstaller: callbackPluginInstaller(func(provider string, req discovery.Constraints) (discovery.PluginMeta, error) {
			t.Fatalf("plugin installer should not have been called for %q", provider)
			return discovery.PluginMeta{}, nil
		}),
	}

	// make our vendor paths
	pluginPath := []string{"a", "b"}
	for _, p := range pluginPath {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// add some dummy providers in our plugin dirs
	for i, name := range []string{
		"terraform-provider-exact_v1.2.3_x4",
		"terraform-provider-greater_than_v2.3.4_x4",
	} {

		if err := ioutil.WriteFile(filepath.Join(pluginPath[i], name), []byte("test bin"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{
		"-plugin-dir", "a",
		"-plugin-dir", "b",
	}
	if code := c.Run(args); code == 0 {
		// should have been an error
		t.Fatalf("bad: \n%s", ui.OutputWriter)
	}
}

// Verify that plugin-dir doesn't prevent discovery of internal providers
func TestInit_pluginWithInternal(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-internal"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{"-plugin-dir", "./"}
	//args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("error: %s", ui.ErrorWriter)
	}
}

func TestInit_012UpgradeNeeded(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-012upgrade"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := cli.NewMockUi()
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			"null": []string{"1.0.0"},
		},
		Dir: m.pluginDir(),
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Errorf("wrong exit status %d; want 0\nerror output:\n%s", code, ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "terraform 0.12upgrade") {
		t.Errorf("doesn't look like we detected the need for config upgrade:\n%s", output)
	}
}

func TestInit_012UpgradeNeededInAutomation(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("init-012upgrade"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := cli.NewMockUi()
	m := Meta{
		testingOverrides:    metaOverridesForProvider(testProvider()),
		Ui:                  ui,
		RunningInAutomation: true,
	}

	installer := &mockProviderInstaller{
		Providers: map[string][]string{
			"null": []string{"1.0.0"},
		},
		Dir: m.pluginDir(),
	}

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Errorf("wrong exit status %d; want 0\nerror output:\n%s", code, ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Run terraform init for this configuration at a shell prompt") {
		t.Errorf("doesn't look like we instructed to run Terraform locally:\n%s", output)
	}
	if strings.Contains(output, "terraform 0.12upgrade") {
		// We don't prompt with an exact command in automation mode, since
		// the upgrade process is interactive and so it cannot be run in
		// automation.
		t.Errorf("looks like we incorrectly gave an upgrade command to run:\n%s", output)
	}
}
