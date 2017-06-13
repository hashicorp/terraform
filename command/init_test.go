package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/mitchellh/cli"
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
	if !strings.Contains(output, "Get: file://") {
		t.Fatalf("doesn't look like get: %s", output)
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
	if !strings.Contains(output, "(update)") {
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
		ui := new(cli.MockUi)
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

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		// Unset
		if err := ioutil.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := new(cli.MockUi)
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

		s := testStateRead(t, filepath.Join(
			DefaultDataDir, DefaultStateFilename))
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
	state := testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "hello" {
		t.Fatalf("bad: %#v", v)
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
	state := testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "hello" {
		t.Fatalf("bad: %#v", v)
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
	state := testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "hello" {
		t.Fatalf("bad: %#v", v)
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

	if _, err := os.Stat(filepath.Join(td, "source", DefaultDataDir, DefaultStateFilename)); err != nil {
		t.Fatalf("err: %s", err)
	}

	// a data directory should not have been added to out working dir
	if _, err := os.Stat(filepath.Join(td, DefaultDataDir)); !os.IsNotExist(err) {
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
		ConfigExtra: map[string]interface{}{"path": "hello"},
		Init:        true,
	}

	b, err := m.backendConfig(opts)
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
	state := testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "hello" {
		t.Fatalf("bad: %#v", v)
	}

	if state.Backend.Hash != b.Hash {
		t.Fatal("mismatched state and config backend hashes")
	}

	if state.Backend.Rehash() != b.Rehash() {
		t.Fatal("mismatched state and config re-hashes")
	}

	// init again and make sure nothing changes
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
	state = testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "hello" {
		t.Fatalf("bad: %#v", v)
	}

	if state.Backend.Hash != b.Hash {
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
	state := testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if v := state.Backend.Config["path"]; v != "foo" {
		t.Fatalf("bad: %#v", v)
	}

	backendHash := state.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := ioutil.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	args := []string{"-input=false", "-backend-config=path=foo"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
	state = testStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))

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

	args = []string{"-input=false", "-backend-config=path=bar"}
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

	c := &InitCommand{
		Meta:              m,
		providerInstaller: installer,
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
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
