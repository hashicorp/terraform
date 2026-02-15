package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	version "github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	httpBackend "github.com/hashicorp/terraform/internal/backend/remote-state/http"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// cleanString removes newlines, and redundant spaces.
func cleanString(s string) string {
	// Replace newlines with a single space.
	s = strings.ReplaceAll(s, "\n", " ")

	// Remove other special characters like \r, \t
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")

	// Replace multiple spaces with a single space.
	spaceRegex := regexp.MustCompile(`\s+`)
	s = spaceRegex.ReplaceAllString(s, " ")

	// Trim any leading or trailing spaces.
	s = strings.TrimSpace(s)

	return s
}

func TestInit_empty(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}
	exp := views.MessageRegistry[views.OutputInitEmptyMessage].JSONValue
	actual := cleanString(done(t).All())
	if !strings.Contains(actual, cleanString(exp)) {
		t.Fatalf("expected output to be %q\n, got %q", exp, actual)
	}
}

func TestInit_only_test_files(t *testing.T) {
	// Create a temporary working directory that has only test files and no tf configuration
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	if _, err := os.Create("main.tftest.hcl"); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}
	exp := views.MessageRegistry[views.OutputInitSuccessCLIMessage].JSONValue
	actual := cleanString(done(t).All())
	if !strings.Contains(actual, cleanString(exp)) {
		t.Fatalf("expected output to be %q\n, got %q", exp, actual)
	}
}

func TestInit_two_step_provider_download(t *testing.T) {
	cases := map[string]struct {
		workDirPath          string
		flags                []string
		expectedDownloadMsgs []string
	}{
		"providers required by only the state file": {
			// TODO - should the output indicate that no providers were found in config?
			workDirPath: "init-provider-download/state-file-only",
			expectedDownloadMsgs: []string{
				views.MessageRegistry[views.OutputInitSuccessCLIMessage].JSONValue,
				`Initializing provider plugins found in the configuration...
				Initializing the backend...`, // No providers found in the configuration so next output is backend-related
				`Initializing provider plugins found in the state...
				- Finding latest version of hashicorp/random...
				- Installing hashicorp/random v9.9.9...`, // The latest version is expected, as state has no version constraints
			},
		},
		"different providers required by config and state": {
			workDirPath: "init-provider-download/config-and-state-different-providers",
			expectedDownloadMsgs: []string{
				views.MessageRegistry[views.OutputInitSuccessCLIMessage].JSONValue,

				// Config - this provider is affected by a version constraint
				`Initializing provider plugins found in the configuration...
				- Finding hashicorp/null versions matching "< 9.0.0"...
				- Installing hashicorp/null v1.0.0...
				- Installed hashicorp/null v1.0.0`,

				// State - the latest version of this provider is expected, as state has no version constraints
				`Initializing provider plugins found in the state...
				- Finding latest version of hashicorp/random...
				- Installing hashicorp/random v9.9.9...`,
			},
		},
		"does not re-download providers that are present in both config and state": {
			workDirPath: "init-provider-download/config-and-state-same-providers",
			expectedDownloadMsgs: []string{
				// Config
				`Initializing provider plugins found in the configuration...
				- Finding hashicorp/random versions matching "< 9.0.0"...
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
				// State
				`Initializing provider plugins found in the state...
				- Reusing previous version of hashicorp/random
				- Using previously-installed hashicorp/random v1.0.0`,
			},
		},
		"reuses providers already represented in a dependency lock file": {
			workDirPath: "init-provider-download/config-state-file-and-lockfile",
			expectedDownloadMsgs: []string{
				// Config
				`Initializing provider plugins found in the configuration...
				- Reusing previous version of hashicorp/random from the dependency lock file
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
				// State
				`Initializing provider plugins found in the state...
				- Reusing previous version of hashicorp/random
				- Using previously-installed hashicorp/random v1.0.0`,
			},
		},
		"using the -upgrade flag causes provider download to ignore the lock file": {
			workDirPath: "init-provider-download/config-state-file-and-lockfile",
			flags:       []string{"-upgrade"},
			expectedDownloadMsgs: []string{
				// Config - lock file is not mentioned due to the -upgrade flag
				`Initializing provider plugins found in the configuration...
				- Finding hashicorp/random versions matching "< 9.0.0"...
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
				// State - reuses the provider download from the config
				`Initializing provider plugins found in the state...
				- Reusing previous version of hashicorp/random
				- Using previously-installed hashicorp/random v1.0.0`,
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Create a temporary working directory no tf configuration but has state
			td := t.TempDir()
			testCopyDir(t, testFixturePath(tc.workDirPath), td)
			os.MkdirAll(td, 0755)
			t.Chdir(td)

			// A provider source containing the random and null providers
			providerSource, close := newMockProviderSource(t, map[string][]string{
				"hashicorp/random": {"1.0.0", "9.9.9"},
				"hashicorp/null":   {"1.0.0", "9.9.9"},
			})
			defer close()

			ui := new(cli.MockUi)
			view, done := testView(t)
			c := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
					ProviderSource:   providerSource,

					AllowExperimentalFeatures: true, // Needed to test init changes for PSS project
				},
			}

			args := append(tc.flags, "-enable-pluggable-state-storage-experiment") // Needed to test init changes for PSS project
			if code := c.Run(args); code != 0 {
				t.Fatalf("bad: \n%s", done(t).All())
			}

			actual := cleanString(done(t).All())
			for _, downloadMsg := range tc.expectedDownloadMsgs {
				if !strings.Contains(cleanString(actual), cleanString(downloadMsg)) {
					t.Fatalf("expected output to contain %q\n, got %q", cleanString(downloadMsg), cleanString(actual))
				}
			}
		})
	}
}

// Test that an error is returned if users provide the removed directory argument, which was replaced with -chdir
// See: https://github.com/hashicorp/terraform/commit/ca23a096d8c48544b9bfc6dbf13c66488f9b6964
func TestInit_multipleArgs(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	expectedMsg := "Did you mean to use -chdir?"
	if !strings.Contains(done(t).All(), expectedMsg) {
		t.Fatalf("expected the error message to include %q as part of protecting against deprecated additional arguments.",
			expectedMsg,
		)
	}
}

func TestInit_migrateStateAndJSON(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-migrate-state=true",
		"-json=true",
	}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("error, -migrate-state and -json should be exclusive: \n%s", testOutput.All())
	}

	// Check output
	checkGoldenReference(t, testOutput, "init-migrate-state-with-json")
}

func TestInit_fromModule_cwdDest(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, os.ModePerm)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-from-module=" + testFixturePath("init"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	if _, err := os.Stat(filepath.Join(td, "hello.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// Regression test to check that Terraform doesn't recursively copy
// a directory when the source module includes the current directory.
// See: https://github.com/hashicorp/terraform/issues/518
func TestInit_fromModule_dstInSrc(t *testing.T) {
	// Change to a temporary directory
	td := t.TempDir()
	t.Chdir(td)

	// Create contents
	// 	.
	// ├── issue518.tf
	// └── foo/
	//     └── (empty)
	if err := os.Mkdir("foo", os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Create("issue518.tf"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Instead of using the -chdir flag, we change directory into the directory foo.
	// 	.
	// ├── issue518.tf
	// └── foo/               << current directory
	//     └── (empty)
	if err := os.Chdir("foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// The path ./.. includes the current directory foo.
	args := []string{
		"-from-module=./..",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	// Assert this outcome
	// 	.
	// ├── issue518.tf
	// └── foo/               << current directory
	//     ├── issue518.tf
	//     └── foo/
	//         └── (empty)
	if _, err := os.Stat(filepath.Join(td, "foo", "issue518.tf")); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := os.Stat(filepath.Join(td, "foo", "foo")); err != nil {
		// Note: originally foo was never copied into itself in this scenario,
		// but behavior changed sometime around when -chdir replaced legacy positional
		// path arguments. We may want to revert to the original behavior in a
		// future major release.
		// See: https://github.com/hashicorp/terraform/pull/38059
		t.Fatalf("err: %s", err)
	}

	// We don't expect foo to be copied into itself multiple times
	_, err := os.Stat(filepath.Join(td, "foo", "foo", "foo"))
	if err == nil {
		t.Fatal("expected directory ./foo/foo/foo to not exist, but it does")
	}
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("unexpected err: %s", err)
	}
}

func TestInit_get(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	// Check output
	output := done(t).Stdout()
	if !strings.Contains(output, "foo in foo") {
		t.Fatalf("doesn't look like we installed module 'foo': %s", output)
	}
}

func TestInit_json(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-json"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	// Check output
	output := done(t)
	checkGoldenReference(t, output, "init-get")
}

func TestInit_getUpgradeModules(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-get=true",
		"-upgrade",
	}
	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("command did not complete successfully:\n%s", testOutput.Stderr())
	}

	// Check output
	if !strings.Contains(testOutput.Stdout(), "Upgrading modules...") {
		t.Fatalf("doesn't look like get upgrade: %s", testOutput.Stdout())
	}
}

// Test initializing a backend from config (new working directory with no pre-existing backend state file).
func TestInit_backend_initFromConfig(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// Test init when the -backend=false flag is present (backend state file is used instead of the config).
func TestInit_backend_initFromState(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-file-change-to-s3"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-backend=false",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	// Double check that the successful init above was due to ignoring the config.
	// When we don't provide -backend=false there should be an error due to a config change being detected;
	// the config specifies an s3 backend instead of local.
	args = []string{}
	view, done = testView(t)
	c.View = view
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad, expected a 'Backend configuration changed' error but command succeeded : \n%s", done(t).All())
	}
}

// regression test for https://github.com/hashicorp/terraform/issues/38027
func TestInit_backend_migration_stateMgr_error(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	t.Chdir(td)

	{
		// create some state in (implied) local backend
		outputCfg := `output "test" { value = "test" }
`
		if err := os.WriteFile("output.tf", []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := new(cli.MockUi)
		applyView, done := testView(t)
		applyCmd := &ApplyCommand{
			Meta: Meta{
				Ui:   ui,
				View: applyView,
			},
		}
		code := applyCmd.Run([]string{"-auto-approve"})
		testOut := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOut.All())
		}

		if _, err := os.Stat(DefaultStateFilename); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		// attempt to migrate the state to a broken backend
		testBackend := new(httpBackend.TestHTTPBackend)
		testBackend.SetMethodFunc("GET", func(w http.ResponseWriter, r *http.Request) {
			// simulate "broken backend" in the way described in #38027
			// i.e. access denied
			w.WriteHeader(403)
		})
		ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
		t.Cleanup(ts.Close)

		backendCfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
		if err := os.WriteFile("backend.tf", []byte(backendCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := new(cli.MockUi)
		initView, done := testView(t)
		initCmd := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: initView,
			},
		}
		code := initCmd.Run([]string{"-migrate-state"})
		out := done(t)
		if code == 0 {
			t.Fatalf("expected migration to fail (gracefully): %s", out.Stdout())
		}
		expectedErrMsg := "HTTP remote state endpoint invalid auth"
		if !strings.Contains(out.Stderr(), expectedErrMsg) {
			t.Fatalf("expected error %q, given: %s", expectedErrMsg, out.Stderr())
		}

		getCalled := testBackend.CallCount("GET")
		if getCalled != 1 {
			t.Fatalf("expected GET to be called exactly %d, called %d times", 1, getCalled)
		}
	}
}

func TestInit_backendUnset(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	{
		log.Printf("[TRACE] TestInit_backendUnset: beginning first init")

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}

		// Init
		args := []string{}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_backendUnset: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		log.Printf("[TRACE] TestInit_backendUnset: beginning second init")

		// Unset
		if err := os.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}

		args := []string{"-force-copy"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_backendUnset: second init complete")
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if !s.Backend.Empty() {
			t.Fatal("should not have backend config")
		}
	}
}

func TestInit_backendConfigFile(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-file"), td)
	t.Chdir(td)

	t.Run("good-config-file", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}
		args := []string{"-backend-config", "input.config"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", done(t).All())
		}

		// Read our saved backend config and verify we have our settings
		state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
	})

	// the backend config file must not be a full terraform block
	t.Run("full-backend-config-file", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}
		args := []string{"-backend-config", "backend.config"}
		if code := c.Run(args); code != 1 {
			t.Fatalf("expected error, got success\n")
		}
		if !strings.Contains(done(t).All(), "Unsupported block type") {
			t.Fatalf("wrong error: %s", done(t).Stderr())
		}
	})

	// the backend config file must match the schema for the backend
	t.Run("invalid-config-file", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}
		args := []string{"-backend-config", "invalid.config"}
		if code := c.Run(args); code != 1 {
			t.Fatalf("expected error, got success\n")
		}
		if !strings.Contains(done(t).All(), "Unsupported argument") {
			t.Fatalf("wrong error: %s", done(t).Stderr())
		}
	})

	// missing file is an error
	t.Run("missing-config-file", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}
		args := []string{"-backend-config", "missing.config"}
		if code := c.Run(args); code != 1 {
			t.Fatalf("expected error, got success\n")
		}
		if !strings.Contains(done(t).All(), "Failed to read file") {
			t.Fatalf("wrong error: %s", done(t).Stderr())
		}
	})

	// blank filename clears the backend config
	t.Run("blank-config-file", func(t *testing.T) {
		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}
		args := []string{"-backend-config=", "-migrate-state"}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", done(t).All())
		}

		// Read our saved backend config and verify the backend config is empty
		state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":null,"workspace_dir":null}`; got != want {
			t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
		}
	})

	// simulate the local backend having a required field which is not
	// specified in the override file
	t.Run("required-argument", func(t *testing.T) {
		c := &InitCommand{}
		schema := &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"path": {
					Type:     cty.String,
					Optional: true,
				},
				"workspace_dir": {
					Type:     cty.String,
					Required: true,
				},
			},
		}
		flagConfigExtra := arguments.NewFlagNameValueSlice("-backend-config")
		flagConfigExtra.Set("input.config")
		_, diags := c.backendConfigOverrideBody(flagConfigExtra, schema)
		if len(diags) != 0 {
			t.Errorf("expected no diags, got: %s", diags.Err())
		}
	})
}

func TestInit_backendConfigFilePowershellConfusion(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-file"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// SUBTLE: when using -flag=value with Powershell, unquoted values are
	// broken into separate arguments. This results in the init command
	// interpreting the flags as an empty backend-config setting (which is
	// semantically valid!) followed by a custom configuration path.
	//
	// Adding the "=" here forces this codepath to be checked, and it should
	// result in an early exit with a diagnostic that the provided
	// configuration file is not a diretory.
	args := []string{"-backend-config=", "./input.config"}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if got, want := output.Stderr(), `Too many command line arguments`; !strings.Contains(got, want) {
		t.Fatalf("wrong output\ngot:\n%s\n\nwant: message containing %q", got, want)
	}
}

func TestInit_backendReconfigure(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			ProviderSource:   providerSource,
			Ui:               ui,
			View:             view,
		},
	}

	// create some state, so the backend has something to migrate.
	f, err := os.Create("foo") // this is the path" in the backend config
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = writeStateForTesting(testState(), f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// now run init again, changing the path.
	// The -reconfigure flag prevents init from migrating
	// Without -reconfigure, the test fails since the backend asks for input on migrating state
	args = []string{"-reconfigure", "-backend-config", "path=changed"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}
}

func TestInit_backendConfigFileChange(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-file-change"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-backend-config", "input.config", "-migrate-state"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestInit_backendMigrateWhileLocked(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-migrate-while-locked"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			ProviderSource:   providerSource,
			Ui:               ui,
			View:             view,
		},
	}

	// Create some state, so the backend has something to migrate from
	f, err := os.Create("local-state.tfstate")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = writeStateForTesting(testState(), f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Lock the source state
	unlock, err := testLockState(t, testDataDir, "local-state.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	// Attempt to migrate
	args := []string{"-backend-config", "input.config", "-migrate-state", "-force-copy"}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected nonzero exit code: %s", done(t).Stdout())
	}

	// Disabling locking should work
	args = []string{"-backend-config", "input.config", "-migrate-state", "-force-copy", "-lock=false"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("expected zero exit code, got %d: %s", code, done(t).Stderr())
	}
}

func TestInit_backendConfigFileChangeWithExistingState(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-file-change-migrate-existing"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, _ := testView(t)

	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	oldState := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))

	// we deliberately do not provide the answer for backend-migrate-copy-to-empty to trigger error
	args := []string{"-migrate-state", "-backend-config", "input.config", "-input=true"}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected error")
	}

	// Read our backend config and verify new settings are not saved
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"local-state.tfstate"}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}

	// without changing config, hash should not change
	if oldState.Backend.Hash != state.Backend.Hash {
		t.Errorf("backend hash should not have changed\ngot:  %d\nwant: %d", state.Backend.Hash, oldState.Backend.Hash)
	}
}

func TestInit_backendConfigKV(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-kv"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-backend-config", "path=hello"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"hello","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}
}

func TestInit_backendConfigKVReInit(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-config-kv"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-backend-config", "path=test"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// a second init should require no changes, nor should it change the backend.
	args = []string{"-input=false"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// make sure the backend is configured how we expect
	configState := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	cfg := map[string]interface{}{}
	if err := json.Unmarshal(configState.Backend.ConfigRaw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["path"] != "test" {
		t.Fatalf(`expected backend path="test", got path="%v"`, cfg["path"])
	}

	// override the -backend-config options by settings
	args = []string{"-input=false", "-backend-config", "", "-migrate-state"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// make sure the backend is configured how we expect
	configState = testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	cfg = map[string]interface{}{}
	if err := json.Unmarshal(configState.Backend.ConfigRaw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["path"] != nil {
		t.Fatalf(`expected backend path="<nil>", got path="%v"`, cfg["path"])
	}
}

func TestInit_backendConfigKVReInitWithConfigDiff(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-input=false"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// a second init with identical config should require no changes, nor
	// should it change the backend.
	args = []string{"-input=false", "-backend-config", "path=foo"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// make sure the backend is configured how we expect
	configState := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	cfg := map[string]interface{}{}
	if err := json.Unmarshal(configState.Backend.ConfigRaw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["path"] != "foo" {
		t.Fatalf(`expected backend path="foo", got path="%v"`, cfg["foo"])
	}
}

func TestInit_backendCli_no_config_block(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-backend-config", "path=test"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("got exit status %d; want 0\nstderr:\n%s\n\nstdout:\n%s", code, done(t).Stderr(), done(t).Stdout())
	}

	errMsg := done(t).All()
	if !strings.Contains(errMsg, "Warning: Missing backend configuration") {
		t.Fatal("expected missing backend block warning, got", errMsg)
	}
}

func TestInit_backendReinitWithExtra(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend-empty"), td)
	t.Chdir(td)

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
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-backend-config", "path=hello"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
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
		t.Fatalf("bad: \n%s", done(t).Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	if code := c.Run([]string{"-input=false"}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// Read our saved backend config and verify we have our settings
	state := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"foo","workspace_dir":null}`; got != want {
		t.Errorf("wrong config\ngot:  %s\nwant: %s", got, want)
	}

	backendHash := state.Backend.Hash

	// init again but remove the path option from the config
	cfg := "terraform {\n  backend \"local\" {}\n}\n"
	if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// We need a fresh InitCommand here because the old one now has our configuration
	// file cached inside it, so it won't re-read the modification we just made.
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-input=false", "-backend-config=path=foo"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}
	state = testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
	if got, want := normalizeJSON(t, state.Backend.ConfigRaw), `{"path":"foo","workspace_dir":null}`; got != want {
		t.Errorf("wrong config after moving to arg\ngot:  %s\nwant: %s", got, want)
	}

	if state.Backend.Hash == backendHash {
		t.Fatal("state.Backend.Hash was not updated")
	}
}

func TestInit_backendCloudInvalidOptions(t *testing.T) {
	// There are various "terraform init" options that are only for
	// traditional backends and not applicable to HCP Terraform mode.
	// For those, we want to return an explicit error rather than
	// just silently ignoring them, so that users will be aware that
	// Cloud mode has more of an expected "happy path" than the
	// less-vertically-integrated backends do, and to avoid these
	// unapplicable options becoming compatibility constraints for
	// future evolution of Cloud mode.

	// We use the same starting fixture for all of these tests, but some
	// of them will customize it a bit as part of their work.
	setupTempDir := func(t *testing.T) {
		t.Helper()
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-cloud-simple"), td)
		t.Chdir(td)
	}

	// Some of the tests need a non-empty placeholder state file to work
	// with.
	fakeState := states.BuildState(func(cb *states.SyncState) {
		// Having a root module output value should be enough for this
		// state file to be considered "non-empty" and thus a candidate
		// for migration.
		cb.SetOutputValue(
			addrs.OutputValue{Name: "a"}.Absolute(addrs.RootModuleInstance),
			cty.True,
			false,
		)
	})
	fakeStateFile := &statefile.File{
		Lineage:          "boop",
		Serial:           4,
		TerraformVersion: version.Must(version.NewVersion("1.0.0")),
		State:            fakeState,
	}
	var fakeStateBuf bytes.Buffer
	err := statefile.WriteForTest(fakeStateFile, &fakeStateBuf)
	if err != nil {
		t.Error(err)
	}
	fakeStateBytes := fakeStateBuf.Bytes()

	t.Run("-backend-config", func(t *testing.T) {
		setupTempDir(t)

		// We have -backend-config as a pragmatic way to dynamically set
		// certain settings of backends that tend to vary depending on
		// where Terraform is running, such as AWS authentication profiles
		// that are naturally local only to the machine where Terraform is
		// running. Those needs don't apply to HCP Terraform, because
		// the remote workspace encapsulates all of the details of how
		// operations and state work in that case, and so the Cloud
		// configuration is only about which workspaces we'll be working
		// with.
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-backend-config=anything"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -backend-config=... command line option is only for state backends, and
is not applicable to HCP Terraform-based configurations.

To change the set of workspaces associated with this configuration, edit the
Cloud configuration block in the root module.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-reconfigure", func(t *testing.T) {
		setupTempDir(t)

		// The -reconfigure option was originally imagined as a way to force
		// skipping state migration when migrating between backends, but it
		// has a historical flaw that it doesn't work properly when the
		// initial situation is the implicit local backend with a state file
		// present. The HCP Terraform migration path has some additional
		// steps to take care of more details automatically, and so
		// -reconfigure doesn't really make sense in that context, particularly
		// with its design bug with the handling of the implicit local backend.
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-reconfigure"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -reconfigure option is for in-place reconfiguration of state backends
only, and is not needed when changing HCP Terraform settings.

When using HCP Terraform, initialization automatically activates any new
Cloud configuration settings.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-reconfigure when migrating in", func(t *testing.T) {
		setupTempDir(t)

		// We have a slightly different error message for the case where we
		// seem to be trying to migrate to HCP Terraform with existing
		// state or explicit backend already present.

		if err := os.WriteFile("terraform.tfstate", fakeStateBytes, 0644); err != nil {
			t.Fatal(err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-reconfigure"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -reconfigure option is unsupported when migrating to HCP Terraform,
because activating HCP Terraform involves some additional steps.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-migrate-state", func(t *testing.T) {
		setupTempDir(t)

		// In Cloud mode, migrating in or out always proposes migrating state
		// and changing configuration while staying in cloud mode never migrates
		// state, so this special option isn't relevant.
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-migrate-state"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -migrate-state option is for migration between state backends only, and
is not applicable when using HCP Terraform.

State storage is handled automatically by HCP Terraform and so the state
storage location is not configurable.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-migrate-state when migrating in", func(t *testing.T) {
		setupTempDir(t)

		// We have a slightly different error message for the case where we
		// seem to be trying to migrate to HCP Terraform with existing
		// state or explicit backend already present.

		if err := os.WriteFile("terraform.tfstate", fakeStateBytes, 0644); err != nil {
			t.Fatal(err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-migrate-state"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -migrate-state option is for migration between state backends only, and
is not applicable when using HCP Terraform.

HCP Terraform migrations have additional steps, configured by interactive
prompts.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-force-copy", func(t *testing.T) {
		setupTempDir(t)

		// In Cloud mode, migrating in or out always proposes migrating state
		// and changing configuration while staying in cloud mode never migrates
		// state, so this special option isn't relevant.
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-force-copy"}
		if code := c.Run(args); code == 0 {
			t.Fatalf("unexpected success\n%s", done(t).Stdout())
		}

		gotStderr := done(t).Stderr()
		wantStderr := `
Error: Invalid command-line option

The -force-copy option is for migration between state backends only, and is
not applicable when using HCP Terraform.

State storage is handled automatically by HCP Terraform and so the state
storage location is not configurable.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
	t.Run("-force-copy when migrating in", func(t *testing.T) {
		setupTempDir(t)

		// We have a slightly different error message for the case where we
		// seem to be trying to migrate to HCP Terraform with existing
		// state or explicit backend already present.

		if err := os.WriteFile("terraform.tfstate", fakeStateBytes, 0644); err != nil {
			t.Fatal(err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:   ui,
				View: view,
			},
		}
		args := []string{"-force-copy"}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("unexpected success\n%s", testOutput.Stdout())
		}

		gotStderr := testOutput.Stderr()
		wantStderr := `
Error: Invalid command-line option

The -force-copy option is for migration between state backends only, and is
not applicable when using HCP Terraform.

HCP Terraform migrations have additional steps, configured by interactive
prompts.
`
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong error output\n%s", diff)
		}
	})
}

func TestInit_cloudConfigColorTokensProcessed(t *testing.T) {
	// This test verifies that when the error
	// diagnostic detail contains color formatting tokens like [bold] and
	// [reset], they are properly processed
	// by the diagnostic formatter and do not appear as literal text in the
	// output.
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-cloud-no-workspaces"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	args := []string{}
	code := c.Run(args)
	if code == 0 {
		t.Fatalf("expected error, got success\n%s", done(t).Stdout())
	}

	gotStderr := done(t).Stderr()

	expected := `
Error: failed to create backend alias to target "". The hostname is not in the correct format.


Error: Invalid workspaces configuration

  on main.tf line 7, in terraform:
   7:   cloud {

Missing workspace mapping strategy. Either workspace "tags" or "name" is
required.

The 'workspaces' block configures how Terraform CLI maps its workspaces for
this single
configuration to workspaces within an HCP Terraform or Terraform Enterprise
organization. Two strategies are available:

tags - A set of tags used to select remote HCP Terraform or Terraform
Enterprise workspaces to be used for this single
configuration. New workspaces will automatically be tagged with these tag
values. Generally, this
is the primary and recommended strategy to use.  This option conflicts with
"name".

name - The name of a single HCP Terraform or Terraform Enterprise workspace
to be used with this configuration.
When configured, only the specified workspace can be used. This option
conflicts with "tags"
and with the TF_WORKSPACE environment variable.
`

	if diff := cmp.Diff(gotStderr, expected); diff != "" {
		t.Errorf("unexpected output (-got +expected):\n%s", diff)
	}
}

// make sure inputFalse stops execution on migrate
func TestInit_inputFalse(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-input=false", "-backend-config=path=foo"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
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
			View:             view,
		},
	}

	args = []string{"-input=false", "-backend-config=path=bar", "-migrate-state"}
	if code := c.Run(args); code == 0 {
		t.Fatal("init should have failed", done(t).Stdout())
	}

	errMsg := done(t).All()
	if !strings.Contains(errMsg, "interactive input is disabled") {
		t.Fatal("expected input disabled error, got", errMsg)
	}

	ui = new(cli.MockUi)
	c = &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// A missing input=false should abort rather than loop infinitely
	args = []string{"-backend-config=path=baz"}
	if code := c.Run(args); code == 0 {
		t.Fatal("init should have failed", done(t).Stdout())
	}
}

func TestInit_getProvider(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	overrides := metaOverridesForProvider(testProvider())
	ui := new(cli.MockUi)
	view, done := testView(t)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		// looking for an exact version
		"exact": {"1.2.3"},
		// config requires >= 2.3.3
		"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
		// config specifies
		"between": {"3.4.5", "2.3.4", "1.2.3"},
	})
	defer close()
	m := Meta{
		testingOverrides: overrides,
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{
		"-backend=false", // should be possible to install plugins without backend init
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// check that we got the providers for our config
	exactPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/exact/1.2.3/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(exactPath); os.IsNotExist(err) {
		t.Fatal("provider 'exact' not downloaded")
	}
	greaterThanPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/greater-than/2.3.4/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(greaterThanPath); os.IsNotExist(err) {
		t.Fatal("provider 'greater-than' not downloaded")
	}
	betweenPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/between/2.3.4/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(betweenPath); os.IsNotExist(err) {
		t.Fatal("provider 'between' not downloaded")
	}

	t.Run("future-state", func(t *testing.T) {
		// getting providers should fail if a state from a newer version of
		// terraform exists, since InitCommand.getProviders needs to inspect that
		// state.

		f, err := os.Create(DefaultStateFilename)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		defer f.Close()

		// Construct a mock state file from the far future
		type FutureState struct {
			Version          uint                     `json:"version"`
			Lineage          string                   `json:"lineage"`
			TerraformVersion string                   `json:"terraform_version"`
			Outputs          map[string]interface{}   `json:"outputs"`
			Resources        []map[string]interface{} `json:"resources"`
		}
		fs := &FutureState{
			Version:          999,
			Lineage:          "123-456-789",
			TerraformVersion: "999.0.0",
			Outputs:          make(map[string]interface{}),
			Resources:        make([]map[string]interface{}, 0),
		}
		src, err := json.MarshalIndent(fs, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal future state: %s", err)
		}
		src = append(src, '\n')
		_, err = f.Write(src)
		if err != nil {
			t.Fatal(err)
		}

		ui := new(cli.MockUi)
		view, done := testView(t)
		m.Ui = ui
		m.View = view
		c := &InitCommand{
			Meta: m,
		}

		code := c.Run(nil)
		testOutput := done(t)
		if code == 0 {
			t.Fatal("expected error, got:", testOutput.Stdout())
		}

		errMsg := testOutput.Stderr()
		if !strings.Contains(errMsg, "Unsupported state file format") {
			t.Fatal("unexpected error:", errMsg)
		}
	})
}

func TestInit_getProviderSource(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-provider-source"), td)
	t.Chdir(td)

	overrides := metaOverridesForProvider(testProvider())
	ui := new(cli.MockUi)
	view, done := testView(t)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		// looking for an exact version
		"acme/alpha": {"1.2.3"},
		// config doesn't specify versions for other providers
		"registry.example.com/acme/beta": {"1.0.0"},
		"gamma":                          {"2.0.0"},
	})
	defer close()
	m := Meta{
		testingOverrides: overrides,
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{
		"-backend=false", // should be possible to install plugins without backend init
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	// check that we got the providers for our config
	exactPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/acme/alpha/1.2.3/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(exactPath); os.IsNotExist(err) {
		t.Error("provider 'alpha' not downloaded")
	}
	greaterThanPath := fmt.Sprintf(".terraform/providers/registry.example.com/acme/beta/1.0.0/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(greaterThanPath); os.IsNotExist(err) {
		t.Error("provider 'beta' not downloaded")
	}
	betweenPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/gamma/2.0.0/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(betweenPath); os.IsNotExist(err) {
		t.Error("provider 'gamma' not downloaded")
	}
}

func TestInit_getProviderLegacyFromState(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-provider-legacy-from-state"), td)
	t.Chdir(td)

	overrides := metaOverridesForProvider(testProvider())
	ui := new(cli.MockUi)
	view, done := testView(t)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"acme/alpha": {"1.2.3"},
	})
	defer close()
	m := Meta{
		testingOverrides: overrides,
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}
	code := c.Run(nil)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}

	// Expect this diagnostic output
	wants := []string{
		"Invalid legacy provider address",
		"You must complete the Terraform 0.13 upgrade process",
	}
	got := testOutput.All()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n\n%s", want, got)
		}
	}
}

func TestInit_getProviderInvalidPackage(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-provider-invalid-package"), td)
	t.Chdir(td)

	overrides := metaOverridesForProvider(testProvider())
	ui := new(cli.MockUi)
	view, done := testView(t)

	// create a provider source which allows installing an invalid package
	addr := addrs.MustParseProviderSourceString("invalid/package")
	version := getproviders.MustParseVersion("1.0.0")
	meta, close, err := getproviders.FakeInstallablePackageMeta(
		addr,
		version,
		getproviders.VersionList{getproviders.MustParseVersion("5.0")},
		getproviders.CurrentPlatform,
		"terraform-package", // should be "terraform-provider-package"
	)
	defer close()
	if err != nil {
		t.Fatalf("failed to prepare fake package for %s %s: %s", addr.ForDisplay(), version, err)
	}
	providerSource := getproviders.NewMockSource([]getproviders.PackageMeta{meta}, nil)

	m := Meta{
		testingOverrides: overrides,
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{
		"-backend=false", // should be possible to install plugins without backend init
	}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}

	// invalid provider should be installed
	packagePath := fmt.Sprintf(".terraform/providers/registry.terraform.io/invalid/package/1.0.0/%s/terraform-package", getproviders.CurrentPlatform)
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Fatal("provider 'invalid/package' not downloaded")
	}

	wantErrors := []string{
		"Failed to install provider",
		"could not find executable file starting with terraform-provider-package",
	}
	got := testOutput.All()
	for _, wantError := range wantErrors {
		if !strings.Contains(got, wantError) {
			t.Fatalf("missing error:\nwant: %q\ngot:\n%s", wantError, got)
		}
	}
}

func TestInit_getProviderDetectedLegacy(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-provider-detected-legacy"), td)
	t.Chdir(td)

	// We need to construct a multisource with a mock source and a registry
	// source: the mock source will return ErrRegistryProviderNotKnown for an
	// unknown provider, and the registry source will allow us to look up the
	// appropriate namespace if possible.
	providerSource, psClose := newMockProviderSource(t, map[string][]string{
		"hashicorp/foo":           {"1.2.3"},
		"terraform-providers/baz": {"2.3.4"}, // this will not be installed
	})
	defer psClose()
	registrySource, rsClose := testRegistrySource(t)
	defer rsClose()
	multiSource := getproviders.MultiSource{
		{Source: providerSource},
		{Source: registrySource},
	}

	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		Ui:             ui,
		View:           view,
		ProviderSource: multiSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{
		"-backend=false", // should be possible to install plugins without backend init
	}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("expected error, got output: \n%s", testOutput.Stdout())
	}

	// foo should be installed
	fooPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/foo/1.2.3/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(fooPath); os.IsNotExist(err) {
		t.Error("provider 'foo' not installed")
	}
	// baz should not be installed
	bazPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/terraform-providers/baz/2.3.4/%s", getproviders.CurrentPlatform)
	if _, err := os.Stat(bazPath); !os.IsNotExist(err) {
		t.Error("provider 'baz' installed, but should not be")
	}

	// error output is the main focus of this test
	errOutput := testOutput.All()
	errors := []string{
		"Failed to query available provider packages",
		"Could not retrieve the list of available versions",
		"registry.terraform.io/hashicorp/baz",
		"registry.terraform.io/hashicorp/frob",
	}
	for _, want := range errors {
		if !strings.Contains(errOutput, want) {
			t.Fatalf("expected error %q: %s", want, errOutput)
		}
	}
}

func TestInit_providerSource(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-required-providers"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test":      {"1.2.3", "1.2.4"},
		"test-beta": {"1.2.4"},
		"source":    {"1.2.2", "1.2.3", "1.2.1"},
	})
	defer close()

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("bad: \n%s", testOutput.All())
	}
	if strings.Contains(testOutput.Stdout(), "Terraform has initialized, but configuration upgrades may be needed") {
		t.Fatalf("unexpected \"configuration upgrade\" warning in output")
	}

	cacheDir := m.providerLocalCacheDir()
	gotPackages := cacheDir.AllAvailablePackages()
	wantPackages := map[addrs.Provider][]providercache.CachedProvider{
		addrs.NewDefaultProvider("test"): {
			{
				Provider:   addrs.NewDefaultProvider("test"),
				Version:    getproviders.MustParseVersion("1.2.3"),
				PackageDir: expectedPackageInstallPath("test", "1.2.3", false),
			},
		},
		addrs.NewDefaultProvider("test-beta"): {
			{
				Provider:   addrs.NewDefaultProvider("test-beta"),
				Version:    getproviders.MustParseVersion("1.2.4"),
				PackageDir: expectedPackageInstallPath("test-beta", "1.2.4", false),
			},
		},
		addrs.NewDefaultProvider("source"): {
			{
				Provider:   addrs.NewDefaultProvider("source"),
				Version:    getproviders.MustParseVersion("1.2.3"),
				PackageDir: expectedPackageInstallPath("source", "1.2.3", false),
			},
		},
	}
	if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
		t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
	}

	locks, err := m.lockedDependencies()
	if err != nil {
		t.Fatalf("failed to get locked dependencies: %s", err)
	}
	gotProviderLocks := locks.AllProviders()
	wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
		addrs.NewDefaultProvider("test-beta"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("test-beta"),
			getproviders.MustParseVersion("1.2.4"),
			getproviders.MustParseVersionConstraints("= 1.2.4"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("see6W06w09Ea+AobFJ+mbvPTie6ASqZAAdlFZbs8BSM="),
			},
		),
		addrs.NewDefaultProvider("test"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("test"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("= 1.2.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		),
		addrs.NewDefaultProvider("source"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("source"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("= 1.2.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("myS3qb3px3tRBq1ZWRYJeUH+kySWpBc0Yy8rw6W7/p4="),
			},
		),
	}

	if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
		t.Errorf("wrong version selections after upgrade\n%s", diff)
	}

	if got, want := testOutput.Stdout(), "Installed hashicorp/test v1.2.3 (verified checksum)"; !strings.Contains(got, want) {
		t.Fatalf("unexpected output: %s\nexpected to include %q", got, want)
	}
	if got, want := testOutput.All(), "\n  - hashicorp/source\n  - hashicorp/test\n  - hashicorp/test-beta"; !strings.Contains(got, want) {
		t.Fatalf("wrong error message\nshould contain: %s\ngot:\n%s", want, got)
	}
}

func TestInit_cancelModules(t *testing.T) {
	// This test runs `terraform init` as if SIGINT (or similar on other
	// platforms) were sent to it, testing that it is interruptible.

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-registry-module"), td)
	t.Chdir(td)

	// Our shutdown channel is pre-closed so init will exit as soon as it
	// starts a cancelable portion of the process.
	shutdownCh := make(chan struct{})
	close(shutdownCh)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ShutdownCh:       shutdownCh,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded; wanted error\n%s", testOutput.Stdout())
	}

	if got, want := testOutput.Stderr(), `Module installation was canceled by an interrupt signal`; !strings.Contains(got, want) {
		t.Fatalf("wrong error message\nshould contain: %s\ngot:\n%s", want, got)
	}
}

func TestInit_cancelProviders(t *testing.T) {
	// This test runs `terraform init` as if SIGINT (or similar on other
	// platforms) were sent to it, testing that it is interruptible.

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-required-providers"), td)
	t.Chdir(td)

	// Use a provider source implementation which is designed to hang indefinitely,
	// to avoid a race between the closed shutdown channel and the provider source
	// operations.
	providerSource := &getproviders.HangingSource{}

	// Our shutdown channel is pre-closed so init will exit as soon as it
	// starts a cancelable portion of the process.
	shutdownCh := make(chan struct{})
	close(shutdownCh)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
		ShutdownCh:       shutdownCh,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded; wanted error\n%s", testOutput.All())
	}
	// Currently the first operation that is cancelable is provider
	// installation, so our error message comes from there. If we
	// make the earlier steps cancelable in future then it'd be
	// expected for this particular message to change.
	if got, want := testOutput.Stderr(), `Provider installation was canceled by an interrupt signal`; !strings.Contains(got, want) {
		t.Fatalf("wrong error message\nshould contain: %s\ngot:\n%s", want, got)
	}
}

func TestInit_getUpgradePlugins(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		// looking for an exact version
		"exact": {"1.2.3"},
		// config requires >= 2.3.3
		"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
		// config specifies > 1.0.0 , < 3.0.0
		"between": {"3.4.5", "2.3.4", "1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	installFakeProviderPackages(t, &m, map[string][]string{
		"exact":        {"0.0.1"},
		"greater-than": {"2.3.3"},
	})

	c := &InitCommand{
		Meta: m,
	}

	args := []string{
		"-upgrade=true",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("command did not complete successfully:\n%s", done(t).All())
	}

	cacheDir := m.providerLocalCacheDir()
	gotPackages := cacheDir.AllAvailablePackages()
	wantPackages := map[addrs.Provider][]providercache.CachedProvider{
		// "between" wasn't previously installed at all, so we installed
		// the newest available version that matched the version constraints.
		addrs.NewDefaultProvider("between"): {
			{
				Provider:   addrs.NewDefaultProvider("between"),
				Version:    getproviders.MustParseVersion("2.3.4"),
				PackageDir: expectedPackageInstallPath("between", "2.3.4", false),
			},
		},
		// The existing version of "exact" did not match the version constraints,
		// so we installed what the configuration selected as well.
		addrs.NewDefaultProvider("exact"): {
			{
				Provider:   addrs.NewDefaultProvider("exact"),
				Version:    getproviders.MustParseVersion("1.2.3"),
				PackageDir: expectedPackageInstallPath("exact", "1.2.3", false),
			},
			// Previous version is still there, but not selected
			{
				Provider:   addrs.NewDefaultProvider("exact"),
				Version:    getproviders.MustParseVersion("0.0.1"),
				PackageDir: expectedPackageInstallPath("exact", "0.0.1", false),
			},
		},
		// The existing version of "greater-than" _did_ match the constraints,
		// but a newer version was available and the user specified
		// -upgrade and so we upgraded it anyway.
		addrs.NewDefaultProvider("greater-than"): {
			{
				Provider:   addrs.NewDefaultProvider("greater-than"),
				Version:    getproviders.MustParseVersion("2.3.4"),
				PackageDir: expectedPackageInstallPath("greater-than", "2.3.4", false),
			},
			// Previous version is still there, but not selected
			{
				Provider:   addrs.NewDefaultProvider("greater-than"),
				Version:    getproviders.MustParseVersion("2.3.3"),
				PackageDir: expectedPackageInstallPath("greater-than", "2.3.3", false),
			},
		},
	}
	if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
		t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
	}

	locks, err := m.lockedDependencies()
	if err != nil {
		t.Fatalf("failed to get locked dependencies: %s", err)
	}
	gotProviderLocks := locks.AllProviders()
	wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
		addrs.NewDefaultProvider("between"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("between"),
			getproviders.MustParseVersion("2.3.4"),
			getproviders.MustParseVersionConstraints("> 1.0.0, < 3.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("JVqAvZz88A+hS2wHVtTWQkHaxoA/LrUAz0H3jPBWPIA="),
			},
		),
		addrs.NewDefaultProvider("exact"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("exact"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("= 1.2.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("H1TxWF8LyhBb6B4iUdKhLc/S9sC/jdcrCykpkbGcfbg="),
			},
		),
		addrs.NewDefaultProvider("greater-than"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("greater-than"),
			getproviders.MustParseVersion("2.3.4"),
			getproviders.MustParseVersionConstraints(">= 2.3.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("SJPpXx/yoFE/W+7eCipjJ+G21xbdnTBD7lWodZ8hWkU="),
			},
		),
	}
	if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
		t.Errorf("wrong version selections after upgrade\n%s", diff)
	}
}

func TestInit_getProviderMissing(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		// looking for exact version 1.2.3
		"exact": {"1.2.4"},
		// config requires >= 2.3.3
		"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
		// config specifies
		"between": {"3.4.5", "2.3.4", "1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("expected error, got output: \n%s", testOutput.Stdout())
	}

	if !strings.Contains(testOutput.All(), "no available releases match") {
		t.Fatalf("unexpected error output: %s", testOutput.Stderr())
	}
}

func TestInit_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-check-required-version"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, done(t).Stderr(), done(t).Stdout())
	}
	errStr := done(t).All()
	if !strings.Contains(errStr, `required_version = "~> 0.9.0"`) {
		t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
	}
	if strings.Contains(errStr, `required_version = ">= 0.13.0"`) {
		t.Fatalf("output should not point to met version constraint, but is:\n\n%s", errStr)
	}
}

// Verify that init will error out with an invalid version constraint, even if
// there are other invalid configuration constructs.
func TestInit_checkRequiredVersionFirst(t *testing.T) {
	t.Run("root_module", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-check-required-version-first"), td)
		t.Chdir(td)

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}

		args := []string{}
		if code := c.Run(args); code != 1 {
			t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, done(t).Stderr(), done(t).Stdout())
		}
		errStr := done(t).All()
		if !strings.Contains(errStr, `Unsupported Terraform Core version`) {
			t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
		}
	})
	t.Run("sub_module", func(t *testing.T) {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-check-required-version-first-module"), td)
		t.Chdir(td)

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
			},
		}

		args := []string{}
		if code := c.Run(args); code != 1 {
			t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, done(t).Stderr(), done(t).Stdout())
		}
		errStr := done(t).All()
		if !strings.Contains(errStr, `Unsupported Terraform Core version`) {
			t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
		}
	})
}

func TestInit_providerLockFile(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-provider-lock-file"), td)
	// The temporary directory does not have write permission (dr-xr-xr-x) after the copy
	defer os.Chmod(td, os.ModePerm)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.2.3"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	lockFile := ".terraform.lock.hcl"
	buf, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read dependency lock file %s: %s", lockFile, err)
	}
	buf = bytes.TrimSpace(buf)
	// The hash in here is for the fake package that newMockProviderSource produces
	// (so it'll change if newMockProviderSource starts producing different contents)
	wantLockFile := strings.TrimSpace(`
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version     = "1.2.3"
  constraints = "1.2.3"
  hashes = [
    "h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
  ]
}
`)
	if diff := cmp.Diff(wantLockFile, string(buf)); diff != "" {
		t.Errorf("wrong dependency lock file contents\n%s", diff)
	}

	// Make the local directory read-only, and verify that rerunning init
	// succeeds, to ensure that we don't try to rewrite an unchanged lock file
	os.Chmod(".", 0555)
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}
}

func TestInit_providerLockFileReadonly(t *testing.T) {
	// The hash in here is for the fake package that newMockProviderSource produces
	// (so it'll change if newMockProviderSource starts producing different contents)
	inputLockFile := strings.TrimSpace(`
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version     = "1.2.3"
  constraints = "1.2.3"
  hashes = [
    "zh:e919b507a91e23a00da5c2c4d0b64bcc7900b68d43b3951ac0f6e5d80387fbdc",
  ]
}
`)

	badLockFile := strings.TrimSpace(`
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version     = "1.2.3"
  constraints = "1.2.3"
  hashes = [
    "zh:0000000000000000000000000000000000000000000000000000000000000000",
  ]
}
`)

	updatedLockFile := strings.TrimSpace(`
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version     = "1.2.3"
  constraints = "1.2.3"
  hashes = [
    "h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
    "zh:e919b507a91e23a00da5c2c4d0b64bcc7900b68d43b3951ac0f6e5d80387fbdc",
  ]
}
`)

	emptyUpdatedLockFile := strings.TrimSpace(`
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.
`)

	cases := []struct {
		desc      string
		fixture   string
		providers map[string][]string
		input     string
		args      []string
		ok        bool
		want      string
	}{
		{
			desc:      "default",
			fixture:   "init-provider-lock-file",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     inputLockFile,
			args:      []string{},
			ok:        true,
			want:      updatedLockFile,
		},
		{
			desc:      "unused provider",
			fixture:   "init-provider-now-unused",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     inputLockFile,
			args:      []string{},
			ok:        true,
			want:      emptyUpdatedLockFile,
		},
		{
			desc:      "readonly",
			fixture:   "init-provider-lock-file",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     inputLockFile,
			args:      []string{"-lockfile=readonly"},
			ok:        true,
			want:      inputLockFile,
		},
		{
			desc:      "unused provider readonly",
			fixture:   "init-provider-now-unused",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     inputLockFile,
			args:      []string{"-lockfile=readonly"},
			ok:        false,
			want:      inputLockFile,
		},
		{
			desc:      "conflict",
			fixture:   "init-provider-lock-file",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     inputLockFile,
			args:      []string{"-lockfile=readonly", "-upgrade"},
			ok:        false,
			want:      inputLockFile,
		},
		{
			desc:      "checksum mismatch",
			fixture:   "init-provider-lock-file",
			providers: map[string][]string{"test": {"1.2.3"}},
			input:     badLockFile,
			args:      []string{"-lockfile=readonly"},
			ok:        false,
			want:      badLockFile,
		},
		{
			desc:    "reject to change required provider dependences",
			fixture: "init-provider-lock-file-readonly-add",
			providers: map[string][]string{
				"test": {"1.2.3"},
				"foo":  {"1.0.0"},
			},
			input: inputLockFile,
			args:  []string{"-lockfile=readonly"},
			ok:    false,
			want:  inputLockFile,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Create a temporary working directory and copy in test fixtures
			td := t.TempDir()
			testCopyDir(t, testFixturePath(tc.fixture), td)
			t.Chdir(td)

			providerSource, close := newMockProviderSource(t, tc.providers)
			defer close()

			ui := new(cli.MockUi)
			view, done := testView(t)
			m := Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
				ProviderSource:   providerSource,
			}

			c := &InitCommand{
				Meta: m,
			}

			// write input lockfile
			lockFile := ".terraform.lock.hcl"
			if err := os.WriteFile(lockFile, []byte(tc.input), 0644); err != nil {
				t.Fatalf("failed to write input lockfile: %s", err)
			}

			code := c.Run(tc.args)
			if tc.ok && code != 0 {
				t.Fatalf("bad: \n%s", done(t).Stderr())
			}
			if !tc.ok && code == 0 {
				t.Fatalf("expected error, got output: \n%s", done(t).Stdout())
			}

			buf, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("failed to read dependency lock file %s: %s", lockFile, err)
			}
			buf = bytes.TrimSpace(buf)
			if diff := cmp.Diff(tc.want, string(buf)); diff != "" {
				t.Errorf("wrong dependency lock file contents\n%s", diff)
			}
		})
	}
}

func TestInit_pluginDirReset(t *testing.T) {
	td := testTempDir(t)
	defer os.RemoveAll(td)
	t.Chdir(td)

	// An empty provider source
	providerSource, close := newMockProviderSource(t, nil)
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
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
		t.Fatalf("bad: \n%s", done(t).Stderr())
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
			View:             view,
			ProviderSource:   providerSource, // still empty
		},
	}

	// make sure we remove the plugin-dir record
	args = []string{"-plugin-dir="}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
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
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	// An empty provider source
	providerSource, close := newMockProviderSource(t, nil)
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	// make our vendor paths
	pluginPath := []string{"a", "b", "c"}
	for _, p := range pluginPath {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// We'll put some providers in our plugin dirs. To do this, we'll pretend
	// for a moment that they are provider cache directories just because that
	// allows us to lean on our existing test helper functions to do this.
	for i, def := range [][]string{
		{"exact", "1.2.3"},
		{"greater-than", "2.3.4"},
		{"between", "2.3.4"},
	} {
		name, version := def[0], def[1]
		dir := providercache.NewDir(pluginPath[i])
		installFakeProviderPackagesElsewhere(t, dir, map[string][]string{
			name: {version},
		})
	}

	args := []string{
		"-plugin-dir", "a",
		"-plugin-dir", "b",
		"-plugin-dir", "c",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}

	locks, err := m.lockedDependencies()
	if err != nil {
		t.Fatalf("failed to get locked dependencies: %s", err)
	}
	gotProviderLocks := locks.AllProviders()
	wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
		addrs.NewDefaultProvider("between"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("between"),
			getproviders.MustParseVersion("2.3.4"),
			getproviders.MustParseVersionConstraints("> 1.0.0, < 3.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("JVqAvZz88A+hS2wHVtTWQkHaxoA/LrUAz0H3jPBWPIA="),
			},
		),
		addrs.NewDefaultProvider("exact"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("exact"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("= 1.2.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("H1TxWF8LyhBb6B4iUdKhLc/S9sC/jdcrCykpkbGcfbg="),
			},
		),
		addrs.NewDefaultProvider("greater-than"): depsfile.NewProviderLock(
			addrs.NewDefaultProvider("greater-than"),
			getproviders.MustParseVersion("2.3.4"),
			getproviders.MustParseVersionConstraints(">= 2.3.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("SJPpXx/yoFE/W+7eCipjJ+G21xbdnTBD7lWodZ8hWkU="),
			},
		),
	}
	if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
		t.Errorf("wrong version selections after upgrade\n%s", diff)
	}

	// -plugin-dir overrides the normal provider source, so it should not have
	// seen any calls at all.
	if calls := providerSource.CallLog(); len(calls) > 0 {
		t.Errorf("unexpected provider source calls (want none)\n%s", spew.Sdump(calls))
	}
}

// Test user-supplied -plugin-dir doesn't allow auto-install
func TestInit_pluginDirProvidersDoesNotGet(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	// Our provider source has a suitable package for "between" available,
	// but we should ignore it because -plugin-dir is set and thus this
	// source is temporarily overridden during install.
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"between": {"2.3.4"},
	})
	defer close()

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	// make our vendor paths
	pluginPath := []string{"a", "b"}
	for _, p := range pluginPath {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// We'll put some providers in our plugin dirs. To do this, we'll pretend
	// for a moment that they are provider cache directories just because that
	// allows us to lean on our existing test helper functions to do this.
	for i, def := range [][]string{
		{"exact", "1.2.3"},
		{"greater-than", "2.3.4"},
	} {
		name, version := def[0], def[1]
		dir := providercache.NewDir(pluginPath[i])
		installFakeProviderPackagesElsewhere(t, dir, map[string][]string{
			name: {version},
		})
	}

	args := []string{
		"-plugin-dir", "a",
		"-plugin-dir", "b",
	}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		// should have been an error
		t.Fatalf("succeeded; want error\nstdout:\n%s\nstderr\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	// The error output should mention the "between" provider but should not
	// mention either the "exact" or "greater-than" provider, because the
	// latter two are available via the -plugin-dir directories.
	errStr := testOutput.Stderr()
	if subStr := "hashicorp/between"; !strings.Contains(errStr, subStr) {
		t.Errorf("error output should mention the 'between' provider\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "hashicorp/exact"; strings.Contains(errStr, subStr) {
		t.Errorf("error output should not mention the 'exact' provider\ndo not want substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "hashicorp/greater-than"; strings.Contains(errStr, subStr) {
		t.Errorf("error output should not mention the 'greater-than' provider\ndo not want substr: %s\ngot:\n%s", subStr, errStr)
	}

	if calls := providerSource.CallLog(); len(calls) > 0 {
		t.Errorf("unexpected provider source calls (want none)\n%s", spew.Sdump(calls))
	}
}

// Verify that plugin-dir doesn't prevent discovery of internal providers
func TestInit_pluginDirWithBuiltIn(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-internal"), td)
	t.Chdir(td)

	// An empty provider source
	providerSource, close := newMockProviderSource(t, nil)
	defer close()

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	args := []string{"-plugin-dir", "./"}
	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("error: %s", testOutput.Stderr())
	}

	outputStr := testOutput.Stdout()
	if subStr := "terraform.io/builtin/terraform is built in to Terraform"; !strings.Contains(outputStr, subStr) {
		t.Errorf("output should mention the terraform provider\nwant substr: %s\ngot:\n%s", subStr, outputStr)
	}
}

func TestInit_invalidBuiltInProviders(t *testing.T) {
	// This test fixture includes two invalid provider dependencies:
	// - an implied dependency on terraform.io/builtin/terraform with an
	//   explicit version number, which is not allowed because it's builtin.
	// - an explicit dependency on terraform.io/builtin/nonexist, which does
	//   not exist at all.
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-internal-invalid"), td)
	t.Chdir(td)

	// An empty provider source
	providerSource, close := newMockProviderSource(t, nil)
	defer close()

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	c := &InitCommand{
		Meta: m,
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded, but was expecting error\nstdout:\n%s\nstderr:\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	errStr := testOutput.Stderr()
	if subStr := "Cannot use terraform.io/builtin/terraform: built-in"; !strings.Contains(errStr, subStr) {
		t.Errorf("error output should mention the terraform provider\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Cannot use terraform.io/builtin/nonexist: this Terraform release"; !strings.Contains(errStr, subStr) {
		t.Errorf("error output should mention the 'nonexist' provider\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
}

func TestInit_invalidSyntaxNoBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-syntax-invalid-no-backend"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		Ui:   ui,
		View: view,
	}

	c := &InitCommand{
		Meta: m,
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded, but was expecting error\nstdout:\n%s\nstderr:\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	errStr := testOutput.Stderr()
	if subStr := "Terraform encountered problems during initialisation, including problems\nwith the configuration, described below."; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should include preamble\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Unsupported block type"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention the syntax problem\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
}

func TestInit_invalidSyntaxWithBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-syntax-invalid-with-backend"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		Ui:   ui,
		View: view,
	}

	c := &InitCommand{
		Meta: m,
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded, but was expecting error\nstdout:\n%s\nstderr:\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	errStr := testOutput.Stderr()
	if subStr := "Terraform encountered problems during initialisation, including problems\nwith the configuration, described below."; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should include preamble\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Unsupported block type"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention the syntax problem\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
}

func TestInit_invalidSyntaxInvalidBackend(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-syntax-invalid-backend-invalid"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		Ui:   ui,
		View: view,
	}

	c := &InitCommand{
		Meta: m,
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded, but was expecting error\nstdout:\n%s\nstderr:\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	errStr := testOutput.Stderr()
	if subStr := "Terraform encountered problems during initialisation, including problems\nwith the configuration, described below."; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should include preamble\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Unsupported block type"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention syntax errors\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Unsupported backend type"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention the invalid backend\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
}

func TestInit_invalidSyntaxBackendAttribute(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-syntax-invalid-backend-attribute-invalid"), td)
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	m := Meta{
		Ui:   ui,
		View: view,
	}

	c := &InitCommand{
		Meta: m,
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("succeeded, but was expecting error\nstdout:\n%s\nstderr:\n%s", testOutput.Stdout(), testOutput.Stderr())
	}

	errStr := testOutput.All()
	if subStr := "Terraform encountered problems during initialisation, including problems\nwith the configuration, described below."; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should include preamble\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Invalid character"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention the invalid character\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
	if subStr := "Error: Invalid expression"; !strings.Contains(errStr, subStr) {
		t.Errorf("Error output should mention the invalid expression\nwant substr: %s\ngot:\n%s", subStr, errStr)
	}
}

func TestInit_testsWithExternalProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-tests-external-providers"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/testing": {"1.0.0"},
		"testing/configure": {"1.0.0"},
	})
	defer close()

	hashicorpTestingProviderAddress := addrs.NewDefaultProvider("testing")
	hashicorpTestingProvider := new(testing_provider.MockProvider)
	testingConfigureProviderAddress := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "testing", "configure")
	testingConfigureProvider := new(testing_provider.MockProvider)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					hashicorpTestingProviderAddress: providers.FactoryFixed(hashicorpTestingProvider),
					testingConfigureProviderAddress: providers.FactoryFixed(testingConfigureProvider),
				},
			},
			Ui:             ui,
			View:           view,
			ProviderSource: providerSource,
		},
	}

	var args []string
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}
}

func TestInit_tests(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-tests"), td)
	t.Chdir(td)

	provider := applyFixtureProvider() // We just want the types from this provider.

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", done(t).Stderr())
	}
}

func TestInit_testsWithProvider(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-tests-with-provider"), td)
	t.Chdir(td)

	provider := applyFixtureProvider() // We just want the types from this provider.

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("expected failure but got: \n%s", testOutput.All())
	}

	got := testOutput.Stderr()
	want := `
Error: Failed to query available provider packages

Could not retrieve the list of available versions for provider
hashicorp/test: no available releases match the given constraints 1.0.1,
1.0.2

To see which modules are currently depending on hashicorp/test and what
versions are specified, run the following command:
    terraform providers
`
	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Fatalf("wrong error message: \ngot:\n%s\nwant:\n%s\ndiff:\n%s", got, want, diff)
	}
}

func TestInit_testsWithOverriddenInvalidRequiredProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-overrides-and-duplicates"), td)
	t.Chdir(td)

	provider := applyFixtureProvider() // We just want the types from this provider.

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	code := c.Run(args) // just make sure it doesn't crash.
	if code != 1 {
		t.Fatalf("expected failure but got: \n%s", done(t).All())
	}
}

func TestInit_testsWithInvalidRequiredProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-duplicates"), td)
	t.Chdir(td)

	provider := applyFixtureProvider() // We just want the types from this provider.

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	code := c.Run(args) // just make sure it doesn't crash.
	if code != 1 {
		t.Fatalf("expected failure but got: \n%s", done(t).All())
	}
}

func TestInit_testsWithModule(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-with-tests-with-module"), td)
	t.Chdir(td)

	provider := applyFixtureProvider() // We just want the types from this provider.

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("bad: \n%s", testOutput.All())
	}

	// Check output
	output := testOutput.Stdout()
	if !strings.Contains(output, "test.main.setup in setup") {
		t.Fatalf("doesn't look like we installed the test module': %s", output)
	}
}

// Testing init's behaviors with `state_store` when run in an empty working directory
func TestInit_stateStore_newWorkingDir(t *testing.T) {
	t.Run("the init command creates a backend state file, and creates the default workspace by default", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			// The test fixture config has no version constraints, so the latest version will
			// be used; below is the 'latest' version in the test world.
			"hashicorp/test": {"1.2.3"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{"-enable-pluggable-state-storage-experiment=true"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			"Initializing the state store...",
			"Terraform created an empty state file for the default workspace",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert the default workspace was created
		if _, exists := mockProvider.MockStates[backend.DefaultStateName]; !exists {
			t.Fatal("expected the default workspace to be created during init, but it is missing")
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}
		v1_2_3, _ := version.NewVersion("1.2.3")
		expectedState := &workdir.StateStoreConfigState{
			Type:      "test_store",
			ConfigRaw: []byte("{\n      \"value\": \"foobar\"\n    }"),
			Hash:      uint64(4158988729),
			Provider: &workdir.ProviderConfigState{
				Version: v1_2_3,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n        \"region\": null\n      }"),
			},
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	t.Run("an init command with the flag -create-default-workspace=false will not make the default workspace by default", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		args := []string{"-enable-pluggable-state-storage-experiment=true", "-create-default-workspace=false"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutput := `Terraform has been configured to skip creation of the default workspace`
		if !strings.Contains(output, expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, output)
		}

		// Assert the default workspace was created
		if _, exists := mockProvider.MockStates[backend.DefaultStateName]; exists {
			t.Fatal("expected Terraform to skip creating the default workspace, but it has been created")
		}
	})

	t.Run("an init command with TF_SKIP_CREATE_DEFAULT_WORKSPACE set will not make the default workspace by default", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource: providerSource,
			},
		}

		t.Setenv("TF_SKIP_CREATE_DEFAULT_WORKSPACE", "1") // any value
		args := []string{"-enable-pluggable-state-storage-experiment=true"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutput := `Terraform has been configured to skip creation of the default workspace`
		if !strings.Contains(output, expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, output)
		}

		// Assert the default workspace was created
		if _, exists := mockProvider.MockStates[backend.DefaultStateName]; exists {
			t.Fatal("expected Terraform to skip creating the default workspace, but it has been created")
		}
	})

	// This scenario would be rare, but protecting against it is easy and avoids assumptions.
	t.Run("if a custom workspace is selected but no workspaces exist an error is returned", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		// Select a custom workspace (which will not exist)
		customWorkspace := "my-custom-workspace"
		t.Setenv(WorkspaceNameEnvVar, customWorkspace)

		mockProvider := mockPluggableStateStorageProvider()
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{"-enable-pluggable-state-storage-experiment=true"}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			fmt.Sprintf("Workspace %q has not been created yet", customWorkspace),
			fmt.Sprintf("To create the custom workspace %q use the command `terraform workspace new %s`", customWorkspace, customWorkspace),
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(cleanString(output), expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, cleanString(output))
			}
		}

		// Assert no workspaces exist
		if len(mockProvider.MockStates) != 0 {
			t.Fatalf("expected no workspaces, but got: %#v", mockProvider.MockStates)
		}

		// Assert no backend state file made due to the error
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		_, err := os.Stat(statePath)
		if pathErr, ok := err.(*os.PathError); !ok || !os.IsNotExist(pathErr.Err) {
			t.Fatalf("expected backend state file to not be created, but it exists")
		}
	})

	// Test what happens when the selected workspace doesn't exist, but there are other workspaces available.
	//
	// When input is disabled (in automation, etc) Terraform cannot prompts the user to select an alternative.
	// Instead, an error is returned.
	t.Run("init: returns an error when input is disabled and the selected workspace doesn't exist and other custom workspaces do exist.", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{
			States: []string{
				"foobar1",
				"foobar2",
				// Force provider to report workspaces exist
				// But default workspace doesn't exist
			},
		}

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// If input is disabled users receive an error about the missing workspace
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-input=false",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}
		output := testOutput.All()
		expectedOutput := "Failed to select a workspace: Currently selected workspace \"default\" does not exist"
		if !strings.Contains(cleanString(output), expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, cleanString(output))
		}
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		_, err := os.Stat(statePath)
		if _, ok := err.(*os.PathError); !ok {
			if err == nil {
				t.Fatalf("expected backend state file to not be created, but it exists")
			}

			t.Fatalf("unexpected error: %s", err)
		}
	})

	// Test what happens when the selected workspace doesn't exist, but there are other workspaces available.
	//
	// When input is enabled Terraform prompts the user to select an alternative.
	t.Run("init: prompts user to select a workspace if the selected workspace doesn't exist and other custom workspaces do exist.", func(t *testing.T) {
		// Create a temporary, uninitialized working directory with configuration including a state store
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-with-state-store"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{
			States: []string{
				"foobar1",
				"foobar2",
				// Force provider to report workspaces exist
				// But default workspace doesn't exist
			},
		}

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		// Allow the test to respond to the prompt to pick an
		// existing workspace, given the selected one doesn't exist.
		defer testInputMap(t, map[string]string{
			"select-workspace": "1", // foobar1 in numbered list
		})()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// The init command should have caused the selected workspace to change, based on the input
		// provided by the user.
		currentWorkspace, err := c.Meta.Workspace()
		if err != nil {
			t.Fatal(err)
		}
		if currentWorkspace != "foobar1" {
			t.Fatalf("expected init command to alter the selected workspace from 'default' to 'foobar1', but got: %s", currentWorkspace)
		}
	})

	// TODO(SarahFrench/radeksimko): Add test cases below:
	// 1) "during a non-init command, the command ends in with an error telling the user to run an init command"
	// >>> Currently this is handled at a lower level in `internal/command/meta_backend_test.go`
}

// Testing init's behaviors with `state_store` when run in a working directory where the configuration
// doesn't match the backend state file.
func TestInit_stateStore_configUnchanged(t *testing.T) {
	// This matches the backend state test fixture in "state-store-unchanged"
	v1_2_3, _ := version.NewVersion("1.2.3")
	expectedState := &workdir.StateStoreConfigState{
		Type:      "test_store",
		ConfigRaw: []byte("{\n            \"value\": \"foobar\"\n        }"),
		Hash:      uint64(4158988729),
		Provider: &workdir.ProviderConfigState{
			Version: v1_2_3,
			Source: &tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "test",
			},
			ConfigRaw: []byte("{\n                \"region\": null\n            }"),
		},
	}

	t.Run("init is successful when the configuration and backend state match", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that matches the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		// If the working directory was previously initialized successfully then at least
		// one workspace is guaranteed to exist when a user is re-running init with no config
		// changes since last init. So this test says `default` exists.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{
			States: []string{"default"},
		}
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		// Before running init, confirm the contents of the backend state file before
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be present, but there isn't one")
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}

		// Run init command
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			"Initializing the state store...",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Confirm init was a no-op and backend state is unchanged afterwards
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s = sMgr.State()
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})
}

// Testing init's behaviors with `state_store` when run in a working directory where the configuration
// doesn't match the backend state file.
func TestInit_stateStore_configChanges(t *testing.T) {
	t.Run("the -reconfigure flag makes Terraform ignore the backend state file during initialization", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()

		// The previous init implied by this test scenario would have created this.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}}
		mockProvider.MockStates = map[string]interface{}{"default": []byte(`{"version": 4,"terraform_version":"1.15.0","serial": 1,"lineage": "","outputs": {},"resources": [],"checks":[]}`)}

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-reconfigure",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutputs := []string{
			"Initializing the state store...",
			"Terraform has been successfully initialized!",
		}
		for _, expected := range expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("expected output to include %q, but got':\n %s", expected, output)
			}
		}

		// Assert contents of the backend state file
		statePath := filepath.Join(meta.DataDir(), DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil {
			t.Fatal("expected backend state file to be created, but there isn't one")
		}
		v1_2_3, _ := version.NewVersion("1.2.3")
		expectedState := &workdir.StateStoreConfigState{
			Type:      "test_store",
			ConfigRaw: []byte("{\n      \"value\": \"changed-value\"\n    }"),
			Hash:      uint64(1157855489), // The new hash after reconfiguring; this doesn't match the backend state test fixture
			Provider: &workdir.ProviderConfigState{
				Version: v1_2_3,
				Source: &tfaddr.Provider{
					Hostname:  tfaddr.DefaultProviderRegistryHost,
					Namespace: "hashicorp",
					Type:      "test",
				},
				ConfigRaw: []byte("{\n        \"region\": null\n      }"),
			},
		}
		if diff := cmp.Diff(s.StateStore, expectedState); diff != "" {
			t.Fatalf("unexpected diff in backend state file's description of state store:\n%s", diff)
		}
	})

	t.Run("the -backend=false flag makes Terraform ignore config and use only the the backend state file during initialization", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()

		// The previous init implied by this test scenario would have created this.
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}}
		mockProvider.MockStates = map[string]interface{}{"default": []byte(`{"version": 4,"terraform_version":"1.15.0","serial": 1,"lineage": "","outputs": {},"resources": [],"checks":[]}`)}

		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-backend=false",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("expected code 0 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedOutput := "Terraform has been successfully initialized!"
		if !strings.Contains(output, expectedOutput) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedOutput, output)
		}

		// When -backend=false the backend/state store isn't initialized, so we don't expect this
		// output if the flag has the expected effect on Terraform.
		unexpectedOutput := "Initializing the state store..."
		if strings.Contains(output, unexpectedOutput) {
			t.Fatalf("output included %q, which is unexpected if -backend=false is behaving correctly':\n %s", unexpectedOutput, output)
		}
	})

	t.Run("handling changed state store config is currently unimplemented", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/store-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}} // The previous init implied by this test scenario would have created the default workspace.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedMsg := "Changing a state store configuration is not implemented yet"
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedMsg, output)
		}
	})

	t.Run("handling changed state store provider config is currently unimplemented", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-config"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}} // The previous init implied by this test scenario would have created the default workspace.
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedMsg := "Changing a state store configuration is not implemented yet"
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedMsg, output)
		}
	})

	t.Run("handling changed state store type in the same provider is currently unimplemented", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/state-store-type"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		storeName := "test_store"
		otherStoreName := "test_otherstore"
		// Make the provider report that it contains a 2nd storage implementation with the above name
		mockProvider.GetProviderSchemaResponse.StateStores[otherStoreName] = mockProvider.GetProviderSchemaResponse.StateStores[storeName]
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedMsg := "Changing a state store configuration is not implemented yet"
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedMsg, output)
		}
	})

	t.Run("handling changing the provider used for state storage is currently unimplemented", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-used"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProvider.GetStatesResponse = &providers.GetStatesResponse{States: []string{"default"}} // The previous init implied by this test scenario would have created the default workspace.

		// Make a mock that implies its name is test2 based on returned schemas
		mockProvider2 := mockPluggableStateStorageProvider()
		mockProvider2.GetProviderSchemaResponse.StateStores["test2_store"] = mockProvider.GetProviderSchemaResponse.StateStores["test_store"]
		delete(mockProvider2.GetProviderSchemaResponse.StateStores, "test_store")

		mockProviderAddress := addrs.NewDefaultProvider("test")
		mockProviderAddress2 := addrs.NewDefaultProvider("test2")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test":  {"1.2.3"}, // Provider in backend state file fixture
			"hashicorp/test2": {"1.2.3"}, // Provider now used in config
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress:  providers.FactoryFixed(mockProvider),  // test provider
					mockProviderAddress2: providers.FactoryFixed(mockProvider2), // test2 provider
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedMsg := "Changing a state store configuration is not implemented yet"
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedMsg, output)
		}
	})
}

// Testing init's behaviors with `state_store` when the provider used for state storage in a previous init
// command is updated.
//
// TODO: Add a test case showing that downgrading provider version is ok as long as the schema version hasn't
// changed. We should also have a test demonstrating that downgrades when the schema version HAS changed will fail.
func TestInit_stateStore_providerUpgrade(t *testing.T) {
	t.Run("handling upgrading the provider used for state storage is currently unimplemented", func(t *testing.T) {
		// Create a temporary working directory with state store configuration
		// that doesn't match the backend state file
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-changed/provider-upgraded"), td)
		t.Chdir(td)

		mockProvider := mockPluggableStateStorageProvider()
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.2.3", "9.9.9"}, // 1.2.3 is the version used in the backend state file, 9.9.9 is the version being upgraded to
		})
		defer close()

		ui := new(cli.MockUi)
		view, done := testView(t)
		meta := Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}
		c := &InitCommand{
			Meta: meta,
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-upgrade",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 1 {
			t.Fatalf("expected code 1 exit code, got %d, output: \n%s", code, testOutput.All())
		}

		// Check output
		output := testOutput.All()
		expectedMsg := "Changing a state store configuration is not implemented yet"
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected output to include %q, but got':\n %s", expectedMsg, output)
		}
	})
}

func TestInit_stateStore_unset(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	storeName := "test_store"
	otherStoreName := "test_otherstore"
	// Make the provider report that it contains a 2nd storage implementation with the above name
	mockProvider.GetProviderSchemaResponse.StateStores[otherStoreName] = mockProvider.GetProviderSchemaResponse.StateStores[storeName]
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})
	defer close()

	{
		log.Printf("[TRACE] TestInit_stateStore_unset: beginning first init")

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		// Init
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		log.Printf("[TRACE] TestInit_stateStore_unset: beginning second init")

		// Unset
		if err := os.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset: second init complete")
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if !s.StateStore.Empty() {
			t.Fatal("should not have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected empty Backend config after unsetting state store, found: %#v", s.Backend)
		}
	}
}

func TestInit_stateStore_unset_withoutProviderRequirements(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	storeName := "test_store"
	otherStoreName := "test_otherstore"
	// Make the provider report that it contains a 2nd storage implementation with the above name
	mockProvider.GetProviderSchemaResponse.StateStores[otherStoreName] = mockProvider.GetProviderSchemaResponse.StateStores[storeName]
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})
	defer close()

	{
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: beginning first init")

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		// Init
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: beginning second init")
		// Unset state store and provider requirements
		if err := os.WriteFile("main.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := os.WriteFile("providers.tf", []byte(""), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_unset_withoutProviderRequirements: second init complete")
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if !s.StateStore.Empty() {
			t.Fatal("should not have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected empty Backend config after unsetting state store, found: %#v", s.Backend)
		}
	}
}

func TestInit_stateStore_to_backend(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-state-store"), td)
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"}, // Matches provider version in backend state file fixture
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning first init")
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: first init complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning apply")
		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning uninitialised apply")

		backendCfg := []byte(`terraform {
  backend "http" {
    address = "https://example.com"
  }
}
`)
		if err := os.WriteFile("main.tf", backendCfg, 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		code := cApply.Run([]string{"-auto-approve"})
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected apply to fail: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: apply complete")
		expectedErr := "Backend initialization required"
		if !strings.Contains(testOutput.Stderr(), expectedErr) {
			t.Fatalf("unexpected error, expected %q, given: %q", expectedErr, testOutput.Stderr())
		}

		log.Printf("[TRACE] TestInit_stateStore_to_backend: uninitialised apply complete")
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if _, err := os.Stat(filepath.Join(DefaultDataDir, DefaultStateFilename)); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		log.Printf("[TRACE] TestInit_stateStore_to_backend: beginning second init")

		testBackend := new(httpBackend.TestHTTPBackend)
		ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
		t.Cleanup(ts.Close)

		// Override state store to backend
		backendCfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
		if err := os.WriteFile("main.tf", []byte(backendCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						mockProviderAddress: providers.FactoryFixed(mockProvider),
					},
				},
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-migrate-state",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] TestInit_stateStore_to_backend: second init complete")
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if !s.StateStore.Empty() {
			t.Fatal("should not have StateStore config")
		}
		if s.Backend.Empty() {
			t.Fatalf("expected backend to not be empty")
		}

		data, err := statefile.Read(bytes.NewBuffer(testBackend.Data))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data: %s", diff)
		}

		expectedGetCalls := 6
		if testBackend.CallCount("GET") != expectedGetCalls {
			t.Fatalf("expected %d GET calls, got %d", expectedGetCalls, testBackend.CallCount("GET"))
		}
		expectedPostCalls := 1
		if testBackend.CallCount("POST") != expectedPostCalls {
			t.Fatalf("expected %d POST calls, got %d", expectedPostCalls, testBackend.CallCount("POST"))
		}
	}
}

func TestInit_uninitialized_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	cfg := `terraform {
	  required_providers {
	    test = {
	      source = "hashicorp/test"
	    }
	  }
	  state_store "test_store" {
	    provider "test" {}
	    value = "foobar"
	  }
	}
	`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	ui := cli.NewMockUi()
	view, done := testView(t)
	cApply := &ApplyCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}
	code := cApply.Run([]string{})
	testOutput := done(t)
	if code == 0 {
		t.Fatalf("expected apply to fail: \n%s", testOutput.All())
	}
	log.Printf("[TRACE] TestInit_stateStore_to_backend: uninitialised apply with state store complete")
	expectedErr := `provider registry.terraform.io/hashicorp/test: required by this configuration but no version is selected`
	if !strings.Contains(testOutput.Stderr(), expectedErr) {
		t.Fatalf("unexpected error, expected %q, given: %s", expectedErr, testOutput.Stderr())
	}
}

func TestInit_backend_to_stateStore_singleWorkspace(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	testBackend := new(httpBackend.TestHTTPBackend)
	ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
	t.Cleanup(ts.Close)

	cfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if testBackend.CallCount("POST") != 0 {
			t.Fatalf("expected 0 POST calls after init, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 2 {
			t.Fatalf("expected 2 GET calls after init, got %d", testBackend.CallCount("GET"))
		}
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning apply with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		if testBackend.CallCount("POST") != 1 {
			t.Fatalf("expected 1 POST call after apply, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 5 {
			t.Fatalf("expected 5 GET calls after apply, got %d", testBackend.CallCount("GET"))
		}
		data, err := statefile.Read(bytes.NewBuffer(testBackend.Data))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected backend to be empty")
		}

		rawData, ok := mockProvider.MockStates[backend.DefaultStateName].([]byte)
		if !ok {
			t.Fatalf("expected %q state to exist in %s: %#v",
				backend.DefaultStateName,
				mockProviderAddress,
				mockProvider.MockStates)
		}

		data, err := statefile.Read(bytes.NewBuffer(rawData))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data: %s", diff)
		}
	}
}

// TestInit_backend_to_stateStore_noState tests that given no state
// in the source backend, no state is created in the destination state store
// as a result of the migration.
func TestInit_backend_to_stateStore_noState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	testBackend := new(httpBackend.TestHTTPBackend)
	ts := httptest.NewServer(http.HandlerFunc(testBackend.Handle))
	t.Cleanup(ts.Close)

	cfg := fmt.Sprintf(`terraform {
  backend "http" {
    address = %q
  }
}
`, ts.URL)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}
	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("first init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())

		if testBackend.CallCount("POST") != 0 {
			t.Fatalf("expected 0 POST calls after init, got %d", testBackend.CallCount("POST"))
		}
		if testBackend.CallCount("GET") != 2 {
			t.Fatalf("expected 2 GET calls after init, got %d", testBackend.CallCount("GET"))
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("second init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected backend to be empty")
		}

		if len(mockProvider.MockStates) != 0 {
			t.Fatalf("expected no state to exist in %s: %#v",
				mockProviderAddress,
				mockProvider.MockStates)
		}
	}
}

func TestInit_localBackend_to_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	cfg := `terraform {
  backend "local" {}
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}
	{
		log.Printf("[TRACE] %s: beginning first init with local backend", t.Name())
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("first init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning apply with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		b, err := os.ReadFile(DefaultStateFilename)
		if err != nil {
			t.Fatalf("unable to read state file: %s", err)
		}

		data, err := statefile.Read(bytes.NewBuffer(b))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("second init exited with non-zero code %d:\n%s", code, testOutput.Stderr())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		t.Logf("Second run output:\n%s", testOutput.Stdout())
		t.Logf("Second run errors:\n%s", testOutput.Stderr())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected backend to be empty")
		}

		rawData, ok := mockProvider.MockStates[backend.DefaultStateName].([]byte)
		if !ok {
			t.Fatalf("expected %q state to exist in %s: %#v",
				backend.DefaultStateName,
				mockProviderAddress,
				mockProvider.MockStates)
		}

		data, err := statefile.Read(bytes.NewBuffer(rawData))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data: %s", diff)
		}

		if f, err := os.Stat(DefaultStateFilename); err == nil && f.Size() > 0 {
			t.Fatalf("expected state file to have been removed at %q. Has size %d bytes.", DefaultStateFilename, f.Size())
		}
	}
}

func TestInit_backend_to_stateStore_multipleWorkspaces(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	cfg := `terraform {
  backend "inmem" {}
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		// run apply to ensure state isn't empty
		// to bypass edge case handling which causes empty state to stop migration
		log.Printf("[TRACE] %s: beginning first apply to default workspace with backend", t.Name())

		outputCfg := `output "test" {
  value = "test"
}
`
		if err := os.WriteFile(filepath.Join(td, "output.tf"), []byte(outputCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		data := inmem.ReadState(t, backend.DefaultStateName)
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cSelect := &WorkspaceSelectCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		sCode := cSelect.Run([]string{"-or-create", "second"})
		aTestOutput := aDone(t)
		if sCode != 0 {
			t.Fatalf("unable to select workspace: \n%s", aTestOutput.All())
		}
		t.Logf("Select workspace output:\n%s", aTestOutput.All())
	}
	{
		ui := cli.NewMockUi()
		aView, aDone := testView(t)
		cApply := &ApplyCommand{
			Meta: Meta{
				Ui:                        ui,
				View:                      aView,
				AllowExperimentalFeatures: true,
			},
		}
		aCode := cApply.Run([]string{"-auto-approve"})
		aTestOutput := aDone(t)
		if aCode != 0 {
			t.Fatalf("bad: \n%s", aTestOutput.All())
		}

		t.Logf("Apply output:\n%s", aTestOutput.Stdout())
		t.Logf("Apply errors:\n%s", aTestOutput.Stderr())

		data := inmem.ReadState(t, "second")
		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}
		if diff := cmp.Diff(expectedOutputs, data.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data after apply: %s", diff)
		}
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
			"-force-copy",
			"-migrate-state",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("second init failed: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		t.Logf("Second init output:\n%s", testOutput.All())

		s := testDataStateRead(t, filepath.Join(DefaultDataDir, DefaultStateFilename))
		if s.StateStore.Empty() {
			t.Fatal("should have StateStore config")
		}
		if !s.Backend.Empty() {
			t.Fatalf("expected backend to be empty")
		}

		expectedOutputs := map[string]*states.OutputValue{
			"test": {
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: "test",
					},
				},
				Value: cty.StringVal("test"),
			},
		}

		expectedWorkspaces := []string{"default", "second"}
		ws := slices.Sorted(maps.Keys(mockProvider.MockStates))
		if diff := cmp.Diff(expectedWorkspaces, ws); diff != "" {
			t.Fatalf("unexpected workspaces: %s", diff)
		}

		// check default workspace first
		rawData, ok := mockProvider.MockStates[backend.DefaultStateName].([]byte)
		if !ok {
			t.Fatalf("expected %q state to exist in %s: %#v",
				backend.DefaultStateName,
				mockProviderAddress,
				mockProvider.MockStates)
		}
		stateData, err := statefile.Read(bytes.NewBuffer(rawData))
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectedOutputs, stateData.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data: %s", diff)
		}

		// check second workspace
		rawData2, ok := mockProvider.MockStates["second"].([]byte)
		if !ok {
			t.Fatalf("expected %q state to exist in %s: %#v",
				backend.DefaultStateName,
				mockProviderAddress,
				mockProvider.MockStates)
		}
		stateData2, err := statefile.Read(bytes.NewBuffer(rawData2))
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectedOutputs, stateData2.State.RootOutputValues); diff != "" {
			t.Fatalf("unexpected data: %s", diff)
		}
	}
}

func TestInit_cloud_to_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()

	ts := cloud.TestServerWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/organizations/hashicorp/workspaces/test": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				w.Write([]byte(`{"data":{"id":"ws-TEST","type":"workspaces","attributes":{"allow-destroy-plan":true,"auto-apply":false,"auto-apply-run-trigger":false,"auto-destroy-activity-duration":null,"auto-destroy-at":null,"auto-destroy-status":null,"inherits-project-auto-destroy":true,"created-at":"2022-06-22T14:24:13.836Z","environment":"default","locked":false,"name":"test","queue-all-runs":false,"speculative-enabled":true,"structured-run-output-enabled":true,"terraform-version":"1.10.0","working-directory":null,"global-remote-state":false,"updated-at":"2026-01-29T15:09:18.075Z","resource-count":0,"apply-duration-average":2000,"plan-duration-average":4000,"policy-check-failures":0,"run-failures":0,"workspace-kpis-runs-count":1,"unarchived-workspace-change-requests-count":0,"latest-change-at":"2026-01-29T15:09:17.200Z","operations":true,"execution-mode":"remote","vcs-repo":null,"vcs-repo-identifier":null,"permissions":{"can-update":true,"can-destroy":true,"can-queue-run":true,"can-read-run":true,"can-read-variable":true,"can-update-variable":true,"can-read-state-versions":true,"can-read-state-outputs":true,"can-create-state-versions":true,"can-queue-apply":true,"can-lock":true,"can-unlock":true,"can-force-unlock":true,"can-read-settings":true,"can-manage-tags":true,"can-manage-run-tasks":true,"can-force-delete":true,"can-manage-assessments":true,"can-manage-ephemeral-workspaces":false,"can-read-assessment-results":true,"can-read-change-requests":false,"can-update-change-requests":false,"can-queue-destroy":true},"actions":{"is-destroyable":true},"description":null,"file-triggers-enabled":true,"trigger-prefixes":[],"trigger-patterns":[],"assessments-enabled":false,"last-assessment-result-at":null,"locked-reason":"","source":"terraform","source-name":null,"source-url":null,"tag-names":[],"setting-overwrites":{"execution-mode":true,"agent-pool":true}},"relationships":{"organization":{"data":{"id":"hashicorp","type":"organizations"}},"current-run":{"data":{"id":"run-TEST","type":"runs"},"links":{"related":"/api/v2/runs/run-TEST"}},"latest-run":{"data":{"id":"run-TEST","type":"runs"},"links":{"related":"/api/v2/runs/run-TEST"}},"outputs":{"data":[{"id":"wsout-TEST","type":"workspace-outputs"}],"links":{"related":"/api/v2/workspaces/ws-TEST/current-state-version-outputs"}},"remote-state-consumers":{"links":{"related":"/api/v2/workspaces/ws-TEST/relationships/remote-state-consumers"}},"current-state-version":{"data":{"id":"sv-TEST","type":"state-versions"},"links":{"related":"/api/v2/workspaces/ws-TEST/current-state-version"}},"current-configuration-version":{"data":{"id":"cv-TEST","type":"configuration-versions"},"links":{"related":"/api/v2/configuration-versions/cv-TEST"}},"agent-pool":{"data":null},"readme":{"data":null},"project":{"data":{"id":"prj-TEST","type":"projects"}},"current-assessment-result":{"data":null},"vars":{"data":[]}},"links":{"self":"/api/v2/organizations/hashicorp/workspaces/test","self-html":"/app/hashicorp/workspaces/test"}}}`))
				w.WriteHeader(http.StatusOK)
				return
			}
		},
		"/api/v2/workspaces/ws-TEST/current-state-version": func(w http.ResponseWriter, r *http.Request) {
			hostname := r.URL.Hostname()
			w.Write(fmt.Appendf([]byte{}, `{"data":{"id":"sv-TEST","type":"state-versions","attributes":{"created-at":"2026-01-29T15:09:17.200Z","size":651,"hosted-state-download-url":"%s/api/state-versions/sv-TEST/hosted_state","hosted-json-state-download-url":"%s/api/state-versions/sv-TEST/hosted_json_state","modules":{},"providers":{},"resources-processed":true,"serial":1,"state-version":4,"status":"finalized","terraform-version":"1.10.0","vcs-commit-url":null,"vcs-commit-sha":null,"resources":[],"billable-rum-count":0},"relationships":{"run":{"data":{"id":"run-TEST","type":"runs"}},"rollback-state-version":{"data":null},"created-by":{"data":{"id":"user-TEST","type":"users"},"links":{"self":"/api/v2/users/user-TEST","related":"/api/v2/runs/run-TEST/created-by"}},"workspace":{"data":{"id":"ws-TEST","type":"workspaces"}},"outputs":{"data":[{"id":"wsout-TEST","type":"state-version-outputs"}],"links":{"related":"/api/v2/state-versions/sv-TEST/outputs"}}},"links":{"self":"/api/v2/state-versions/sv-TEST"}}}`, hostname, hostname))
			w.WriteHeader(http.StatusOK)
		},
		"/api/state-versions/sv-TEST/hosted_state": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"version":4,"terraform_version":"1.15.0","serial":1,"lineage":"91adaece-23b3-7bce-0695-5aea537d2fef","outputs":{"test":{"value":"test","type":"string"}},"resources":[],"check_results":null}`))
			w.WriteHeader(http.StatusOK)
		},
	})
	t.Cleanup(ts.Close)
	mockURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	backendInit.Init(testDisco(ts))
	t.Cleanup(func() { backendInit.Init(nil) })

	cfg := fmt.Sprintf(`terraform {
  cloud {
    hostname = %q
    organization = "hashicorp"
    token = "test-token"
    workspaces {
      name = "test"
    }
  }
}
`, mockURL.Host)
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	mockProvider := mockPluggableStateStorageProvider()
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})
	defer close()

	tOverrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			mockProviderAddress: providers.FactoryFixed(mockProvider),
		},
	}

	{
		log.Printf("[TRACE] %s: beginning first init with backend", t.Name())
		// Init
		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Services:                  testDisco(ts),
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}
		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code != 0 {
			t.Fatalf("bad: \n%s", testOutput.All())
		}
		log.Printf("[TRACE] %s: first init complete", t.Name())
		t.Logf("First run output:\n%s", testOutput.Stdout())
		t.Logf("First run errors:\n%s", testOutput.Stderr())
	}
	{
		log.Printf("[TRACE] %s: beginning second init with state store", t.Name())

		ssCfg := `terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}
`
		if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(ssCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		ui := cli.NewMockUi()
		view, done := testView(t)
		c := &InitCommand{
			Meta: Meta{
				Services:                  testDisco(ts),
				testingOverrides:          tOverrides,
				ProviderSource:            providerSource,
				Ui:                        ui,
				View:                      view,
				AllowExperimentalFeatures: true,
			},
		}

		args := []string{
			"-enable-pluggable-state-storage-experiment=true",
		}
		code := c.Run(args)
		testOutput := done(t)
		if code == 0 {
			t.Fatalf("expected migration from cloud to fail: \n%s", testOutput.Stdout())
		}
		log.Printf("[TRACE] %s: second init with state store complete", t.Name())
		expectedMsg := "Migrating state from HCP Terraform or Terraform Enterprise to another backend is not \nyet implemented."
		if !strings.Contains(testOutput.Stderr(), expectedMsg) {
			t.Fatalf("expected error message %q not found: \n%s", expectedMsg, testOutput.Stderr())
		}
	}
}

// Test that config-parsing errors that prevent initialising the pluggable state store are identified and returned
// before Terraform attempts to initialise the store.
//
// These errors include omitting the necessary entry in required_providers, or causing an issue with how require_providers
// is parsed. This test uses the first scenario for simplicity.
func TestInit_configErrorsImpactingStateStore(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)
	cfg1 := `terraform {
  required_providers {
    foobar = {
      source = "hashicorp/foobar"
    }
  }
  state_store "test_store" {
    provider "test" {} # missing from required_providers
    value = "foobar"
  }
}
	`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg1), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := cli.NewMockUi()
	view, done := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	log.Printf("[TRACE] TestInit_configErrorsImpactingStateStore: init start")
	args := []string{"-enable-pluggable-state-storage-experiment"}
	code := initCmd.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("expected apply to fail with code 1, got code %d: \n%s", code, testOutput.All())
	}
	log.Printf("[TRACE] TestInit_configErrorsImpactingStateStore: init complete")
	t.Logf("init output:\n%s", testOutput.Stdout())
	t.Logf("init errors:\n%s", testOutput.Stderr())

	expectedErrs := []string{
		// Pre-amble text that's shown when a config-parsing error occurs during init.
		"Error: Terraform encountered problems during initialisation, including problems with the configuration, described below.",
		// This parsing error previously wouldn't be reported before initialising the backend, so
		// Terraform attempted to use a state store in the missing provider.
		"Error: Missing entry in required_providers",
	}
	for _, e := range expectedErrs {
		if !strings.Contains(cleanString(testOutput.Stderr()), e) {
			t.Fatalf("unexpected error, expected %q, given: %s", e, testOutput.Stderr())
		}
	}
}

// newMockProviderSource is a helper to succinctly construct a mock provider
// source that contains a set of packages matching the given provider versions
// that are available for installation (from temporary local files).
//
// The caller must call the returned close callback once the source is no
// longer needed, at which point it will clean up all of the temporary files
// and the packages in the source will no longer be available for installation.
//
// Provider addresses must be valid source strings, and passing only the
// provider name will be interpreted as a "default" provider under
// registry.terraform.io/hashicorp. If you need more control over the
// provider addresses, pass a full provider source string.
//
// This function also registers providers as belonging to the current platform,
// to ensure that they will be available to a provider installer operating in
// its default configuration.
//
// In case of any errors while constructing the source, this function will
// abort the current test using the given testing.T. Therefore a caller can
// assume that if this function returns then the result is valid and ready
// to use.
func newMockProviderSource(t *testing.T, availableProviderVersions map[string][]string) (source *getproviders.MockSource, close func()) {
	t.Helper()
	var packages []getproviders.PackageMeta
	var closes []func()
	close = func() {
		for _, f := range closes {
			f()
		}
	}
	for source, versions := range availableProviderVersions {
		addr := addrs.MustParseProviderSourceString(source)
		for _, versionStr := range versions {
			version, err := getproviders.ParseVersion(versionStr)
			if err != nil {
				close()
				t.Fatalf("failed to parse %q as a version number for %q: %s", versionStr, addr.ForDisplay(), err)
			}
			meta, close, err := getproviders.FakeInstallablePackageMeta(addr, version, getproviders.VersionList{getproviders.MustParseVersion("5.0")}, getproviders.CurrentPlatform, "")
			if err != nil {
				close()
				t.Fatalf("failed to prepare fake package for %s %s: %s", addr.ForDisplay(), versionStr, err)
			}
			closes = append(closes, close)
			packages = append(packages, meta)
		}
	}

	return getproviders.NewMockSource(packages, nil), close
}

// installFakeProviderPackages installs a fake package for the given provider
// names (interpreted as a "default" provider address) and versions into the
// local plugin cache for the given "meta".
//
// Any test using this must also use t.TempDir and t.Chdir from the testing library
// or some similar mechanism to make sure that it isn't writing directly into a test
// fixture or source directory within the codebase.
//
// If a requested package cannot be installed for some reason, this function
// will abort the test using the given testing.T. Therefore if this function
// returns the caller can assume that the requested providers have been
// installed.
func installFakeProviderPackages(t *testing.T, meta *Meta, providerVersions map[string][]string) {
	t.Helper()

	cacheDir := meta.providerLocalCacheDir()
	installFakeProviderPackagesElsewhere(t, cacheDir, providerVersions)
}

// installFakeProviderPackagesElsewhere is a variant of installFakeProviderPackages
// that will install packages into the given provider cache directory, rather
// than forcing the use of the local cache of the current "Meta".
func installFakeProviderPackagesElsewhere(t *testing.T, cacheDir *providercache.Dir, providerVersions map[string][]string) {
	t.Helper()

	// It can be hard to spot the mistake of forgetting to use t.TempDir and
	// t.Chdir from the testing library before modifying the working directory,
	// so we'll use a simple heuristic here to try to detect that mistake
	// and make a noisy error about it instead.
	wd, err := os.Getwd()
	if err == nil {
		wd = filepath.Clean(wd)
		// If the directory we're in is named "command" or if we're under a
		// directory named "testdata" then we'll assume a mistake and generate
		// an error. This will cause the test to fail but won't block it from
		// running.
		if filepath.Base(wd) == "command" || filepath.Base(wd) == "testdata" || strings.Contains(filepath.ToSlash(wd), "/testdata/") {
			t.Errorf("installFakeProviderPackage may be used only by tests that switch to a temporary working directory, e.g. using t.TempDir and t.Chdir from the testing library")
		}
	}

	for name, versions := range providerVersions {
		addr := addrs.NewDefaultProvider(name)
		for _, versionStr := range versions {
			version, err := getproviders.ParseVersion(versionStr)
			if err != nil {
				t.Fatalf("failed to parse %q as a version number for %q: %s", versionStr, name, err)
			}
			meta, close, err := getproviders.FakeInstallablePackageMeta(addr, version, getproviders.VersionList{getproviders.MustParseVersion("5.0")}, getproviders.CurrentPlatform, "")
			// We're going to install all these fake packages before we return,
			// so we don't need to preserve them afterwards.
			defer close()
			if err != nil {
				t.Fatalf("failed to prepare fake package for %s %s: %s", name, versionStr, err)
			}
			_, err = cacheDir.InstallPackage(context.Background(), meta, nil)
			if err != nil {
				t.Fatalf("failed to install fake package for %s %s: %s", name, versionStr, err)
			}
		}
	}
}

// expectedPackageInstallPath is a companion to installFakeProviderPackages
// that returns the path where the provider with the given name and version
// would be installed and, relatedly, where the installer will expect to
// find an already-installed version.
//
// Just as with installFakeProviderPackages, this function is a shortcut helper
// for "default-namespaced" providers as we commonly use in tests. If you need
// more control over the provider addresses, use functions of the underlying
// getproviders and providercache packages instead.
//
// The result always uses forward slashes, even on Windows, for consistency
// with how the getproviders and providercache packages build paths.
func expectedPackageInstallPath(name, version string, exe bool) string {
	platform := getproviders.CurrentPlatform
	baseDir := ".terraform/providers"
	if exe {
		p := fmt.Sprintf("registry.terraform.io/hashicorp/%s/%s/%s/terraform-provider-%s_%s", name, version, platform, name, version)
		if platform.OS == "windows" {
			p += ".exe"
		}
		return filepath.ToSlash(filepath.Join(baseDir, p))
	}
	return filepath.ToSlash(filepath.Join(
		baseDir, fmt.Sprintf("registry.terraform.io/hashicorp/%s/%s/%s", name, version, platform),
	))
}

func mockPluggableStateStorageProvider() *testing_provider.MockProvider {
	// Create a mock provider to use for PSS
	// Get mock provider factory to be used during init
	//
	// This imagines a provider called `test` that contains
	// a pluggable state store implementation called `store`.
	pssName := "test_store"
	mock := testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"region": {Type: cty.String, Optional: true},
					},
				},
			},
			DataSources: map[string]providers.Schema{},
			ResourceTypes: map[string]providers.Schema{
				"test_instance": {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"input": {Type: cty.String, Optional: true},
							"id":    {Type: cty.String, Computed: true},
						},
					},
				},
			},
			ListResourceTypes: map[string]providers.Schema{},
			StateStores: map[string]providers.Schema{
				pssName: {
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
	mock.ConfigureStateStoreFn = func(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
		return providers.ConfigureStateStoreResponse{
			Capabilities: providers.StateStoreServerCapabilities{
				ChunkSize: 1234, // arbitrary number that isn't 0
			},
		}
	}
	mock.WriteStateBytesFn = func(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
		// Workspaces exist once the artefact representing it is written
		if _, exist := mock.MockStates[req.StateId]; !exist {
			// Ensure non-nil map
			if mock.MockStates == nil {
				mock.MockStates = make(map[string]interface{})
			}
		}
		mock.MockStates[req.StateId] = req.Bytes

		return providers.WriteStateBytesResponse{
			Diagnostics: nil, // success
		}
	}
	mock.ReadStateBytesFn = func(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
		state := []byte{}
		if v, exist := mock.MockStates[req.StateId]; exist {
			state = v.([]byte) // If this panics, the mock has been set up with a bad MockStates value
		}
		return providers.ReadStateBytesResponse{
			Bytes:       state,
			Diagnostics: nil, // success
		}
	}
	return &mock
}
