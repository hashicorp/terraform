package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	version "github.com/hashicorp/go-version"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	httpBackend "github.com/hashicorp/terraform/internal/backend/remote-state/http"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
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
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	ui := testUiWrapped(t)
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
	exp := "Terraform initialized in an empty directory!"
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

	ui := testUiWrapped(t)
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
	exp := "You may now begin working with Terraform"
	actual := cleanString(done(t).All())
	if !strings.Contains(actual, cleanString(exp)) {
		t.Fatalf("expected output to be %q\n, got %q", exp, actual)
	}
}

func TestInit_two_source_provider_download(t *testing.T) {
	cases := map[string]struct {
		workDirPath          string
		flags                []string
		expectedDownloadMsgs []string
	}{
		"providers required by only the state file": {
			workDirPath: "init-provider-download/state-file-only",
			expectedDownloadMsgs: []string{
				"You may now begin working with Terraform",
				`Initializing the backend...
				Successfully configured the backend "local"!`, // No providers found in the configuration so next output is backend-related
				`Initializing provider plugins...
				- Finding latest version of hashicorp/random...
				- Installing hashicorp/random v9.9.9...`, // The latest version is expected, as state has no version constraints
			},
		},
		"different providers required by config and state": {
			workDirPath: "init-provider-download/config-and-state-different-providers",
			expectedDownloadMsgs: []string{
				"You may now begin working with Terraform",
				"Initializing provider plugins...",
				// Config - null provider is affected by a version constraint
				`- Finding hashicorp/null versions matching "< 9.0.0"...`,
				`- Installing hashicorp/null v1.0.0...
				- Installed hashicorp/null v1.0.0`,
				// State - the latest version of random provider is expected, as state has no version constraints
				`- Finding latest version of hashicorp/random...`,
				`- Installing hashicorp/random v9.9.9...
				- Installed hashicorp/random v9.9.9`,
			},
		},
		"does not re-download providers that are present in both config and state": {
			workDirPath: "init-provider-download/config-and-state-same-providers",
			expectedDownloadMsgs: []string{
				`Initializing provider plugins...
				- Finding hashicorp/random versions matching "< 9.0.0"...
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
			},
		},
		"reuses providers already represented in a dependency lock file": {
			workDirPath: "init-provider-download/config-state-file-and-lockfile",
			expectedDownloadMsgs: []string{
				`Initializing provider plugins...
				- Reusing version 1.0.0 of hashicorp/random from the dependency lock file
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
			},
		},
		"using the -upgrade flag causes provider download to ignore the lock file": {
			workDirPath: "init-provider-download/config-state-file-and-lockfile",
			flags:       []string{"-upgrade"},
			expectedDownloadMsgs: []string{
				// lock file is not mentioned due to the -upgrade flag
				`Initializing provider plugins...
				- Finding hashicorp/random versions matching "< 9.0.0"...
				- Installing hashicorp/random v1.0.0...
				- Installed hashicorp/random v1.0.0`,
			},
		},
		// Same as some tests above, but now the version constraint in config specifies a pre-release
		"pre-release not re-downloaded if present in both config and state": {
			workDirPath: "init-provider-download-prerelease/config-and-state-same-providers",
			expectedDownloadMsgs: []string{
				`Initializing provider plugins...
				- Finding hashicorp/random versions matching "1.2.3-beta"...
				- Installing hashicorp/random v1.2.3-beta...
				- Installed hashicorp/random v1.2.3-beta (verified checksum)`,
			},
		},
		"reuses pre-release provider already represented in a dependency lock file": {
			workDirPath: "init-provider-download-prerelease/config-state-file-and-lockfile",
			expectedDownloadMsgs: []string{
				`Initializing provider plugins...
				- Reusing version 1.2.3-beta of hashicorp/random from the dependency lock file
				- Installing hashicorp/random v1.2.3-beta...
				- Installed hashicorp/random v1.2.3-beta`,
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
			providerSource := newMockProviderSource(t, map[string][]string{
				"hashicorp/random": {"1.0.0", "1.2.3-beta", "9.9.9"},
				"hashicorp/null":   {"1.0.0", "1.2.3-beta", "9.9.9"},
			})
			ui := testUiWrapped(t)
			view, done := testView(t)
			c := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
					ProviderSource:   providerSource,
				},
			}

			if code := c.Run(tc.flags); code != 0 {
				t.Fatalf("bad: \n%s", done(t).All())
			}

			actual := done(t).All()
			for _, downloadMsg := range tc.expectedDownloadMsgs {
				if !strings.Contains(cleanString(actual), cleanString(downloadMsg)) {
					t.Fatalf("expected output to contain %q\n, got:\n%s", cleanString(downloadMsg), actual)
				}
			}
		})
	}
}

func TestInit_stateStoreProviderDownload(t *testing.T) {
	cases := map[string]struct {
		workDirPath          string
		flags                []string
		expectedDownloadMsgs []string
	}{
		"does not re-download the provider used for PSS in the second provider download step": {
			workDirPath: "init-provider-download/state-store-config-only",
			flags:       []string{"-enable-pluggable-state-storage-experiment"},
			expectedDownloadMsgs: []string{
				`Installing provider hashicorp/test for state store "test_store"...
				- Finding latest version of hashicorp/test...
				- Installing hashicorp/test v1.2.3...
				- Installed hashicorp/test v1.2.3`,
				`Initializing the state store "test_store"...`,
				`Initializing provider plugins...
				- Reusing version 1.2.3 of hashicorp/test from the dependency lock file
				- Using previously-installed hashicorp/test v1.2.3`,
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
			providerSource := newMockProviderSource(t, map[string][]string{
				"hashicorp/random": {"1.0.0", "1.2.3-beta", "9.9.9"},
				"hashicorp/null":   {"1.0.0", "1.2.3-beta", "9.9.9"},
				"hashicorp/test":   {"1.2.3"},
			})

			mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))

			ui := testUiWrapped(t)
			view, done := testView(t)
			c := &InitCommand{
				Meta: Meta{
					testingOverrides:          metaOverridesForProvider(mockProvider),
					Ui:                        ui,
					View:                      view,
					ProviderSource:            providerSource,
					AllowExperimentalFeatures: true,
				},
			}

			if code := c.Run(tc.flags); code != 0 {
				t.Fatalf("bad: \n%s", done(t).All())
			}

			actual := done(t).All()
			for _, downloadMsg := range tc.expectedDownloadMsgs {
				if !strings.Contains(cleanString(actual), cleanString(downloadMsg)) {
					t.Fatalf("expected output to contain %q\n, got:\n%s", cleanString(downloadMsg), actual)
				}
			}
		})
	}
}

// A lock file is insufficient to use a pre-release, version constraints in config are also needed.
// This is true prior to init being split into two download steps, so we're documenting that behaviour here.
func TestInit_cannotUsePreReleaseWithoutConfigConstraint(t *testing.T) {
	// Create a temporary working directory no tf configuration but has state
	td := t.TempDir()
	workDirPath := "init-provider-download-prerelease/state-and-lock-file"
	testCopyDir(t, testFixturePath(workDirPath), td)
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// A provider source containing the random provider
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/random": {"1.0.0", "1.2.3-beta", "9.9.9"},
	})

	ui := testUiWrapped(t)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("expected exit code 1, got: %d \n%s", code, done(t).All())
	}

	actual := cleanString(done(t).All())
	expectedErrorMsgs := []string{
		`Could not retrieve the list of available versions for provider hashicorp/random: locked provider registry.terraform.io/hashicorp/random 1.2.3-beta does not match configured version constraint `,
	}
	for _, errorMsg := range expectedErrorMsgs {
		if !strings.Contains(cleanString(actual), cleanString(errorMsg)) {
			t.Fatalf("expected output to contain %q\n, got %q", cleanString(errorMsg), cleanString(actual))
		}
	}
}

// Test that an error is returned if users provide the removed directory argument, which was replaced with -chdir
// See: https://github.com/hashicorp/terraform/commit/ca23a096d8c48544b9bfc6dbf13c66488f9b6964
func TestInit_multipleArgs(t *testing.T) {
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
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

	expectedMsgs := []string{
		"Error: No positional arguments are expected",
		"-chdir?",
	}
	output := done(t).All()
	for _, expectedMsg := range expectedMsgs {
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected the error message to include %q as part of protecting against deprecated additional arguments.",
				expectedMsg,
			)
		}
	}
}

func TestInit_migrateStateAndJSON(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		_, diags := c.backendConfigOverrideBody(flagConfigExtra, schema, Backend)
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

	ui := testUiWrapped(t)
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

	if got, want := output.Stderr(), `No positional arguments are expected`; !strings.Contains(got, want) {
		t.Fatalf("wrong output\ngot:\n%s\n\nwant: message containing %q", got, want)
	}
}

func TestInit_backendReconfigure(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui = testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui = testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	m := testMetaBackend(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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
		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui = testUiWrapped(t)
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

	ui = testUiWrapped(t)
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
	ui := testUiWrapped(t)
	view, done := testView(t)
	providerSource := newMockProviderSource(t, map[string][]string{
		// looking for an exact version
		"exact": {"1.2.3"},
		// config requires >= 2.3.3
		"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
		// config specifies
		"between": {"3.4.5", "2.3.4", "1.2.3"},
	})
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

		ui := testUiWrapped(t)
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
	ui := testUiWrapped(t)
	view, done := testView(t)
	providerSource := newMockProviderSource(t, map[string][]string{
		// looking for an exact version
		"acme/alpha": {"1.2.3"},
		// config doesn't specify versions for other providers
		"registry.example.com/acme/beta": {"1.0.0"},
		"gamma":                          {"2.0.0"},
	})
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
	ui := testUiWrapped(t)
	view, done := testView(t)
	providerSource := newMockProviderSource(t, map[string][]string{
		"acme/alpha": {"1.2.3"},
	})
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
	ui := testUiWrapped(t)
	view, done := testView(t)

	// create a provider source which allows installing an invalid package
	addr := addrs.MustParseProviderSourceString("invalid/package")
	version := getproviders.MustParseVersion("1.0.0")
	meta, err := getproviders.FakeInstallablePackageMeta(
		t,
		addr,
		version,
		getproviders.VersionList{getproviders.MustParseVersion("5.0")},
		getproviders.CurrentPlatform,
		"terraform-package", // should be "terraform-provider-package"
	)
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
	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/foo":           {"1.2.3"},
		"terraform-providers/baz": {"2.3.4"}, // this will not be installed
	})
	registrySource, rsClose := testRegistrySource(t)
	defer rsClose()
	multiSource := getproviders.MultiSource{
		{Source: providerSource},
		{Source: registrySource},
	}

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"test":      {"1.2.3", "1.2.4"},
		"test-beta": {"1.2.4"},
		"source":    {"1.2.2", "1.2.3", "1.2.1"},
	})

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

// Test different scenarios when upgrading providers with the -upgrade flag is attempted
func TestInit_getUpgradePlugins(t *testing.T) {
	t.Run("no upgrade if -upgrade=false set, and a lockfile controls installed version", func(t *testing.T) {
		// Create a temporary working directory and copy in test fixtures
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-get-providers-with-lockfile"), td)
		t.Chdir(td)

		providerSource := newMockProviderSource(t, map[string][]string{
			// looking for an exact version
			"exact": {"1.2.3"},
			// config requires >= 2.3.3, lockfile specifies 2.3.3
			"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
			// config specifies > 1.0.0, < 3.0.0, lockfile specifies 2.3.4
			"between": {"3.4.5", "2.9.9", "2.3.4", "1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		}

		// Make Terraform believe there are already some versions of some providers installed
		installFakeProviderPackages(t, &m, map[string][]string{
			"exact":        {"0.0.1"},
			"greater-than": {"2.3.3"},
		})

		c := &InitCommand{
			Meta: m,
		}

		args := []string{
			"-upgrade=false",
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
			// The existing version of "greater-than" _did_ match the constraints.
			// This version is in the lock file and, as we're not performing an upgrade,
			// no other version was downloaded.
			addrs.NewDefaultProvider("greater-than"): {
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
				getproviders.MustParseVersion("2.3.3"),
				getproviders.MustParseVersionConstraints(">= 2.3.3"),
				[]getproviders.Hash{
					getproviders.HashScheme1.New("4VW33E3N9PSejfg3jmsBcDAVRy7D+ri2FtqkgVxVJsc="),
				},
			),
		}
		if diff := cmp.Diff(wantProviderLocks, gotProviderLocks, depsfile.ProviderLockComparer); diff != "" {
			t.Errorf("wrong version selections after upgrade\n%s", diff)
		}
	})

	t.Run("the -upgrade flag allows providers to be upgraded to latest versions matching constraints", func(t *testing.T) {
		// Create a temporary working directory and copy in test fixtures
		td := t.TempDir()
		testCopyDir(t, testFixturePath("init-get-providers"), td)
		t.Chdir(td)

		providerSource := newMockProviderSource(t, map[string][]string{
			// looking for an exact version
			"exact": {"1.2.3"},
			// config requires >= 2.3.3
			"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
			// config specifies > 1.0.0 , < 3.0.0
			"between": {"3.4.5", "2.3.4", "1.2.3"},
		})

		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		}

		// Make Terraform believe there are already versions of the providers installed,
		// these are what we're upgrading from.
		installFakeProviderPackages(t, &m, map[string][]string{
			"exact":        {"0.0.1"},
			"greater-than": {"2.3.3"},
			// no "between" installed previously
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
	})

	t.Run("`init -upgrade` cannot be used to upgrade the state store provider", func(t *testing.T) {
		td := t.TempDir()
		t.Chdir(td)

		// Configuration uses 2 providers, including one used for a state store.
		cfg := `
terraform {

  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "> 1.0.0"
    }
    foobar = {
      source  = "hashicorp/foobar"
      version = "> 1.0.0"
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}`
		if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
			t.Fatalf("failed to write main.tf: %s", err)
		}

		providerSource := newMockProviderSource(t, map[string][]string{
			// config requires > 1.0.0 for each of these
			"test":   {"1.2.3", "9.9.9"},
			"foobar": {"1.2.3", "9.9.9"},
		})

		// Mock provider to act as "hashicorp/test"
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})

		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides:          metaOverridesForProvider(mockProvider),
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		}

		// Make Terraform believe that we already have version 1.2.3 installed in a local cache.
		installFakeProviderPackages(t, &m, map[string][]string{
			"test":   {"1.2.3"},
			"foobar": {"1.2.3"},
		})

		// Create a dependency lock file describing both providers at version 1.2.3, to simulate a previous init with that version.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("test"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		locks.SetProvider(
			addrs.NewDefaultProvider("foobar"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		if err := depsfile.SaveLocksToFile(locks, ".terraform.lock.hcl"); err != nil {
			t.Fatalf("failed to write provider locks file: %s", err)
		}

		c := &InitCommand{
			Meta: m,
		}

		args := []string{
			"-upgrade=true",
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("command was expected to complete successfully, but it did not:\n%s", done(t).All())
		}
		output := done(t).Stdout()
		expectedMsg := "Warning: Cannot upgrade the provider used for state storage during \"terraform init -upgrade\""
		if !strings.Contains(output, expectedMsg) {
			t.Fatalf("expected warning message not found:\n%s", output)
		}

		// What happens in the local cache?
		//
		// Prior to the upgrade v1.2.3 of both providers was present
		// After, v9.9.9 of both have been downloaded.
		cacheDir := m.providerLocalCacheDir()
		gotPackages := cacheDir.AllAvailablePackages()
		wantPackages := map[addrs.Provider][]providercache.CachedProvider{
			addrs.NewDefaultProvider("test"): {
				{
					Provider:   addrs.NewDefaultProvider("test"),
					Version:    getproviders.MustParseVersion("9.9.9"),
					PackageDir: expectedPackageInstallPath("test", "9.9.9", false),
				},
				{
					Provider:   addrs.NewDefaultProvider("test"),
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("test", "1.2.3", false),
				},
			},
			addrs.NewDefaultProvider("foobar"): {
				{
					Provider:   addrs.NewDefaultProvider("foobar"),
					Version:    getproviders.MustParseVersion("9.9.9"),
					PackageDir: expectedPackageInstallPath("foobar", "9.9.9", false),
				},
				{
					Provider:   addrs.NewDefaultProvider("foobar"),
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("foobar", "1.2.3", false),
				},
			},
		}
		if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
			t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
		}

		// hashicorp/test has not been upgraded in the lock file, while hashicorp/foobar was upgraded.
		// This is because hashicorp/test is used for state storage.
		expectedContent := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/foobar" {
  version     = "9.9.9"
  constraints = "> 1.0.0"
  hashes = [
    "h1:cHfrPr8b4Wjf8x014vgi4H3qQ6tOd+vEVchLY0PnuFE=",
  ]
}

provider "registry.terraform.io/hashicorp/test" {
  version     = "1.2.3"
  constraints = "> 1.0.0"
  hashes = [
    "h1:wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno=",
  ]
}`
		assertLockfileContents(t, depsfile.LockFilePath, expectedContent)
	})

	// A dev_override setting stops a provider being upgraded. That behaviour is tested elsewhere.
	// This test is a regression test asserting that checks ensuring the state storage provider wasn't
	// upgraded during `init -upgrade` doesn't panic when the state storage provider is a dev_override.
	//
	// An equivalent test for this scenario where the state storage provider is unmanaged is implemented
	// as an E2E test, as that's the only place unmanaged providers can be used in tests.
	t.Run("no errors if `init -upgrade` is run while the state store provider is a dev_override ", func(t *testing.T) {
		// Create a temporary working directory and copy in test fixtures
		td := t.TempDir()
		t.Chdir(td)

		// Configuration uses a state store and has other provider requirements.
		cfg := `
terraform {

  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "> 1.0.0"
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}`
		if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
			t.Fatalf("failed to write main.tf: %s", err)
		}

		providerSource := newMockProviderSource(t, map[string][]string{
			// config requires > 1.0.0
			"test": {"1.2.3", "9.9.9"},
		})

		// Mock provider to act as "hashicorp/test"
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})
		mockProviderAddress := addrs.NewDefaultProvider("test")
		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides:          metaOverridesForProvider(mockProvider),
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,

			// THIS ALLOWS THE TEST TO MIMIC A SCENARIO WHERE THE PROVIDER IS A DEV OVERRIDE.
			// The mock is still accessed via testingOverrides, but Terraform believes that it's a dev override due to
			// its presence in the ProviderDevOverrides map.
			ProviderDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
				mockProviderAddress: ".",
			},
		}

		// Make Terraform believe that we already have version 1.2.3 installed.
		installFakeProviderPackages(t, &m, map[string][]string{
			"test": {"1.2.3"},
		})
		// Create a dependency lock file describing the hashicorp/test provider at version 1.2.3, to simulate a previous init with that version.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("test"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		if err := depsfile.SaveLocksToFile(locks, ".terraform.lock.hcl"); err != nil {
			t.Fatalf("failed to write provider locks file: %s", err)
		}

		c := &InitCommand{
			Meta: m,
		}

		args := []string{
			"-upgrade=true",
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("expected code 0, but got code %d. \nOutput:\n%s", code, done(t).All())
		}

		// Assert that no providers were upgraded.
		//
		// Unlike other test scenarios, "test" v9.9.9 will not be installed in the cache.
		// This is because the installation process skips installing providers that are
		// dev_overrides. That logic is used in both upgrade and non-upgrade operations.
		cacheDir := m.providerLocalCacheDir()
		gotPackages := cacheDir.AllAvailablePackages()
		wantPackages := map[addrs.Provider][]providercache.CachedProvider{
			mockProviderAddress: {
				// No v9.9.9 entry
				{
					Provider:   mockProviderAddress,
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("test", "1.2.3", false),
				},
			},
		}
		if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
			t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
		}

		// The provider locks should not have changed.
		locks, err := m.lockedDependencies()
		if err != nil {
			t.Fatalf("failed to get locked dependencies: %s", err)
		}
		gotProviderLocks := locks.AllProviders()
		wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
			mockProviderAddress: depsfile.NewProviderLock(
				mockProviderAddress,
				getproviders.MustParseVersion("1.2.3"),
				getproviders.MustParseVersionConstraints("> 1.0.0"),
				[]getproviders.Hash{
					getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
				},
			),
		}
		if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
			t.Errorf("wrong version selections after upgrade\n%s", diff)
		}
	})

	t.Run("`init -upgrade -reconfigure` can be used to upgrade the state store provider", func(t *testing.T) {
		// Create a temporary working directory and copy in test fixtures
		td := t.TempDir()
		t.Chdir(td)

		// Configuration uses a state store and has other provider requirements.
		cfg := `
terraform {

  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "> 1.0.0"
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}`
		if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
			t.Fatalf("failed to write main.tf: %s", err)
		}

		providerSource := newMockProviderSource(t, map[string][]string{
			// config requires > 1.0.0
			"test": {"1.2.3", "9.9.9"},
		})

		// Mock provider to act as "hashicorp/test"
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})

		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides:          metaOverridesForProvider(mockProvider),
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		}

		// Make Terraform believe that we already have version 1.2.3 installed.
		installFakeProviderPackages(t, &m, map[string][]string{
			"test": {"1.2.3"},
		})
		// Create a dependency lock file describing the hashicorp/test provider at version 1.2.3, to simulate a previous init with that version.
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("test"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		if err := depsfile.SaveLocksToFile(locks, ".terraform.lock.hcl"); err != nil {
			t.Fatalf("failed to write provider locks file: %s", err)
		}

		c := &InitCommand{
			Meta: m,
		}

		args := []string{
			"-upgrade=true",
			"-reconfigure",
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("expected command to complete successfully, but it did not:\n%s", done(t).All())
		}
		output := done(t).Stdout()
		expectedMsgs := []string{
			"Installed hashicorp/test v9.9.9",
		}
		for _, msg := range expectedMsgs {
			if !strings.Contains(output, msg) {
				t.Fatalf("expected message %q not found:\n%s", msg, output)
			}
		}

		// Assert the provider was upgraded.
		cacheDir := m.providerLocalCacheDir()
		gotPackages := cacheDir.AllAvailablePackages()
		wantPackages := map[addrs.Provider][]providercache.CachedProvider{
			addrs.NewDefaultProvider("test"): {
				{
					Provider:   addrs.NewDefaultProvider("test"),
					Version:    getproviders.MustParseVersion("9.9.9"),
					PackageDir: expectedPackageInstallPath("test", "9.9.9", false),
				},
				{
					Provider:   addrs.NewDefaultProvider("test"),
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("test", "1.2.3", false),
				},
			},
		}
		if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
			t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
		}

		// The provider locks should have changed.
		locks, err := m.lockedDependencies()
		if err != nil {
			t.Fatalf("failed to get locked dependencies: %s", err)
		}
		gotProviderLocks := locks.AllProviders()
		wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
			addrs.NewDefaultProvider("test"): depsfile.NewProviderLock(
				addrs.NewDefaultProvider("test"),
				getproviders.MustParseVersion("9.9.9"),
				getproviders.MustParseVersionConstraints("> 1.0.0"),
				[]getproviders.Hash{
					getproviders.HashScheme1.New("bRxrXpGHyZagKAC5yZZDB9fWBcdllOsFOZzzbJ1VZ0I="),
				},
			),
		}
		if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
			t.Errorf("wrong version selections after upgrade\n%s", diff)
		}
	})

	t.Run("use of pluggable state storage doesn't block upgrading other providers, if the state store provider is pinned", func(t *testing.T) {
		// Create a temporary working directory and copy in test fixtures
		td := t.TempDir()
		t.Chdir(td)

		// Configuration uses a state store and has other provider requirements.
		cfg := `
terraform {

  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3" # pinned to v1.2.3 to allow other provider upgrades
    }
    foobar = {
      source  = "hashicorp/foobar"
      version = "> 1.0.0"
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}`
		if err := os.WriteFile("main.tf", []byte(cfg), 0644); err != nil {
			t.Fatalf("failed to write main.tf: %s", err)
		}

		providerSource := newMockProviderSource(t, map[string][]string{
			// config requires > 1.0.0
			"test":   {"1.2.3", "9.9.9"},
			"foobar": {"1.2.3", "9.9.9"},
		})
		// Mock provider to act as "hashicorp/test"
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProvider.MockStates = testing_provider.NewMockStateBytesWithStateIds("test_store", []string{"default"})

		ui := testUiWrapped(t)
		view, done := testView(t)
		m := Meta{
			testingOverrides:          metaOverridesForProvider(mockProvider),
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		}

		// Make Terraform believe that we already have version 1.2.3 installed.
		installFakeProviderPackages(t, &m, map[string][]string{
			"test":   {"1.2.3"},
			"foobar": {"1.2.3"},
		})
		// Create a dependency lock file
		locks := depsfile.NewLocks()
		locks.SetProvider(
			addrs.NewDefaultProvider("test"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("1.2.3"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
			},
		)
		locks.SetProvider(
			addrs.NewDefaultProvider("foobar"),
			getproviders.MustParseVersion("1.2.3"),
			getproviders.MustParseVersionConstraints("> 1.0.0"),
			[]getproviders.Hash{
				getproviders.HashScheme1.New("z1PjPOAuP6c16GKfaCTIJwj95cZ9PrOiREixVFeLZYA="),
			},
		)
		if err := depsfile.SaveLocksToFile(locks, ".terraform.lock.hcl"); err != nil {
			t.Fatalf("failed to write provider locks file: %s", err)
		}

		c := &InitCommand{
			Meta: m,
		}

		args := []string{
			"-upgrade=true",
			"-enable-pluggable-state-storage-experiment",
		}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("expected command to succeed, but it did not:\n%s", done(t).All())
		}
		output := done(t).Stdout()
		expectedMsgs := []string{
			"Using previously-installed hashicorp/test v1.2.3",
			"Installed hashicorp/foobar v9.9.9",
			"Terraform has made some changes to the provider dependency selections",
		}
		for _, msg := range expectedMsgs {
			if !strings.Contains(output, msg) {
				t.Fatalf("expected message %q not found:\n%s", msg, output)
			}
		}

		// Assert that the non-PSS provider was upgraded.
		cacheDir := m.providerLocalCacheDir()
		gotPackages := cacheDir.AllAvailablePackages()
		wantPackages := map[addrs.Provider][]providercache.CachedProvider{
			addrs.NewDefaultProvider("foobar"): {
				// Newly downloaded version
				{
					Provider:   addrs.NewDefaultProvider("foobar"),
					Version:    getproviders.MustParseVersion("9.9.9"),
					PackageDir: expectedPackageInstallPath("foobar", "9.9.9", false),
				},
				{
					Provider:   addrs.NewDefaultProvider("foobar"),
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("foobar", "1.2.3", false),
				},
			},
			addrs.NewDefaultProvider("test"): {
				{
					Provider:   addrs.NewDefaultProvider("test"),
					Version:    getproviders.MustParseVersion("1.2.3"),
					PackageDir: expectedPackageInstallPath("test", "1.2.3", false),
				},
			},
		}
		if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
			t.Errorf("wrong cache directory contents after upgrade\n%s", diff)
		}

		// The upgrade process has updated the lock file.
		locks, err := m.lockedDependencies()
		if err != nil {
			t.Fatalf("failed to get locked dependencies: %s", err)
		}
		gotProviderLocks := locks.AllProviders()
		wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
			addrs.NewDefaultProvider("foobar"): depsfile.NewProviderLock(
				addrs.NewDefaultProvider("foobar"),
				getproviders.MustParseVersion("9.9.9"),
				getproviders.MustParseVersionConstraints("> 1.0.0"),
				[]getproviders.Hash{
					getproviders.HashScheme1.New("cHfrPr8b4Wjf8x014vgi4H3qQ6tOd+vEVchLY0PnuFE="),
				},
			),
			addrs.NewDefaultProvider("test"): depsfile.NewProviderLock(
				addrs.NewDefaultProvider("test"),
				getproviders.MustParseVersion("1.2.3"),
				getproviders.MustParseVersionConstraints("1.2.3"),
				[]getproviders.Hash{
					getproviders.HashScheme1.New("wlbEC2mChQZ2hhgUhl6SeVLPP7fMqOFUZAQhQ9GIIno="),
				},
			),
		}
		if diff := cmp.Diff(gotProviderLocks, wantProviderLocks, depsfile.ProviderLockComparer); diff != "" {
			t.Errorf("wrong version selections after upgrade\n%s", diff)
		}
	})
}

func TestInit_getProviderMissing(t *testing.T) {
	// Create a temporary working directory and copy in test fixtures
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get-providers"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		// looking for exact version 1.2.3
		"exact": {"1.2.4"},
		// config requires >= 2.3.3
		"greater-than": {"2.3.4", "2.3.3", "2.3.0"},
		// config specifies
		"between": {"3.4.5", "2.3.4", "1.2.3"},
	})

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

		ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"test": {"1.2.3"},
	})

	ui := testUiWrapped(t)
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

			providerSource := newMockProviderSource(t, tc.providers)

			ui := testUiWrapped(t)
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
	providerSource := newMockProviderSource(t, nil)

	ui := testUiWrapped(t)
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

	ui = testUiWrapped(t)
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
	providerSource := newMockProviderSource(t, nil)

	ui := testUiWrapped(t)
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
	providerSource := newMockProviderSource(t, map[string][]string{
		"between": {"2.3.4"},
	})

	ui := testUiWrapped(t)
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
	providerSource := newMockProviderSource(t, nil)

	ui := testUiWrapped(t)
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
	providerSource := newMockProviderSource(t, nil)

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/testing": {"1.0.0"},
		"testing/configure": {"1.0.0"},
	})

	hashicorpTestingProviderAddress := addrs.NewDefaultProvider("testing")
	hashicorpTestingProvider := new(testing_provider.MockProvider)
	testingConfigureProviderAddress := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "testing", "configure")
	testingConfigureProvider := new(testing_provider.MockProvider)

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := testUiWrapped(t)
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

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	ui := testUiWrapped(t)
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

func TestInit_varValueWithoutConfig(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)
	cfg1 := `terraform {}`

	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg1), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := testUiWrapped(t)
	view, done := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	code := initCmd.Run([]string{"-var", "foo=bar"})
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("expected init to fail with code 1, got code %d: \n%s", code, testOutput.All())
	}

	expectedErr := "Error: Value for undeclared variable"
	if !strings.Contains(cleanString(testOutput.Stderr()), expectedErr) {
		t.Fatalf("unexpected error, expected %q, given: %s", expectedErr, testOutput.Stderr())
	}
}

func TestInit_invalidConfig(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)
	cfg1 := `module "test" { version = "1" }`

	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg1), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	ui := testUiWrapped(t)
	view, done := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	code := initCmd.Run([]string{})
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("expected init to fail with code 1, got code %d: \n%s", code, testOutput.All())
	}

	expectedErr := "Error: Missing required argument"
	if !strings.Contains(cleanString(testOutput.Stderr()), expectedErr) {
		t.Fatalf("unexpected error, expected %q, given: %s", expectedErr, testOutput.Stderr())
	}
}

// newMockProviderSource is a helper to succinctly construct a mock provider
// source that contains a set of packages matching the given provider versions
// that are available for installation (from temporary local files).
//
// This function will automatically close the source at the end of the test/subtest
// using t.Cleanup, so the caller isn't responsible for cleaning up all of the
// temporary files and packages in the source.
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
func newMockProviderSource(t *testing.T, availableProviderVersions map[string][]string) *getproviders.MockSource {
	t.Helper()
	var packages []getproviders.PackageMeta
	for source, versions := range availableProviderVersions {
		addr := addrs.MustParseProviderSourceString(source)
		for _, versionStr := range versions {
			version, err := getproviders.ParseVersion(versionStr)
			if err != nil {
				t.Fatalf("failed to parse %q as a version number for %q: %s", versionStr, addr.ForDisplay(), err)
			}
			meta, err := getproviders.FakeInstallablePackageMeta(t, addr, version, getproviders.VersionList{getproviders.MustParseVersion("5.0")}, getproviders.CurrentPlatform, "")
			if err != nil {
				t.Fatalf("failed to prepare fake package for %s %s: %s", addr.ForDisplay(), versionStr, err)
			}
			packages = append(packages, meta)
		}
	}

	return getproviders.NewMockSource(packages, nil)
}

// newMockProviderSourceViaHTTP returns a mock provider source that will return
// metadata for providers defined by the calling test code. That metadata will
// report that the provider package is available for download via HTTP from a given
// address. That address is built using the address parameter.
//
// The mock HTTP provider source returned by newMockProviderSourceViaHTTP is not
// sufficient for Terraform to complete a provider installation process successfully;
// the provider source will supply Terraform with metadata describing where packages
// can be downloaded from, only. Without an HTTP server that serves files matching the
// metadata returned from this source, Terraform will fail during provider download.
//
// Use newMockProviderSourceUsingTestHttpServer, a helper that sets up a test HTTP server
// to use in combination with this source.
func newMockProviderSourceViaHTTP(t *testing.T, availableProviderVersions map[string][]string, address string) (source *getproviders.MockSource) {
	t.Helper()
	var packages []getproviders.PackageMeta
	for source, versions := range availableProviderVersions {
		addr := addrs.MustParseProviderSourceString(source)
		for _, versionStr := range versions {
			version, err := getproviders.ParseVersion(versionStr)
			if err != nil {
				t.Fatalf("failed to parse %q as a version number for %q: %s", versionStr, addr.ForDisplay(), err)
			}
			meta, err := getproviders.FakePackageMetaViaHTTP(
				t,
				addr,
				version,
				getproviders.VersionList{getproviders.MustParseVersion("5.0")},
				getproviders.CurrentPlatform,
				address)
			if err != nil {
				t.Fatalf("failed to prepare fake package for %s %s: %s", addr.ForDisplay(), versionStr, err)
			}
			packages = append(packages, meta)
		}
	}

	return getproviders.NewMockSource(packages, nil)
}

// newMockProviderSourceUsingTestHttpServer is a helper that returns a mock provider
// source that is paired with a test HTTP server. The provider source will tell Terraform
// that a given provider can be downloaded via HTTP from a given URL, and the test HTTP
// server will enable Terraform to perform that download successfully.
//
// This source is not sufficient for providers to be available to _use_ during a test,
// it is only sufficient to enable a provider installation process to complete successfully.
// If your test expects a provider to be installed and then used, ensure that testOverrides
// provides the actual provider implementations for use during the test.
func newMockProviderSourceUsingTestHttpServer(t *testing.T, availableProviderVersions map[string][]string) *getproviders.MockSource {
	t.Helper()

	// Get un-started server so we can obtain the port it'll run on.
	server := httptest.NewUnstartedServer(nil)

	// Set up mock provider source that mocks installation via HTTP.
	source := newMockProviderSourceViaHTTP(
		t,
		availableProviderVersions,
		server.Listener.Addr().String(),
	)

	// Get all the metadata for all provider versions defined in the availableProviderVersions map.
	// This is needed to enable the http server to serve contents of the correct temporary file.
	var packages []getproviders.PackageMeta
	for pSource, versions := range availableProviderVersions {
		addr := addrs.MustParseProviderSourceString(pSource)
		for _, versionStr := range versions {
			version := getproviders.MustParseVersion(versionStr)
			providerMetadata, err := source.PackageMeta(
				context.Background(),
				addr,
				version,
				getproviders.CurrentPlatform,
			)
			if err != nil {
				t.Fatalf("failed to get provider metadata: %s", err)
			}
			packages = append(packages, providerMetadata)
		}
	}

	// Make Terraform believe it's downloading the provider.
	// Any requests to the test server that aren't for that purpose will cause the test to fail.
	server.Config = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var providerMetadata getproviders.PackageMeta

		// Find the package with a location matching the request URL path.
		// E.g. a request to path "/terraform-provider-test/1.2.3/terraform-provider-test_1.2.3_darwin_arm64.zip"
		//      needs to be matched to a provider in the map with Location "http://<server-address>/terraform-provider-test/1.2.3/terraform-provider-test_1.2.3_darwin_arm64.zip"
		found := false
		for _, p := range packages {
			if strings.HasSuffix(p.Location.String(), r.URL.Path) {
				providerMetadata = p
				found = true
				break
			}
		}
		if !found {
			// Cannot process request if it doesn't match the test setup.
			t.Fatalf("unexpected URL path, test doesn't define a matching provider version: %s", r.URL.Path)
		}

		// This code returns data in the temporary file that's created by this test helper.
		// This 'download' is not used when Terraform uses the provider after the mock installation completes;
		// Terraform will look for will use testOverrides in the Meta set up for this test.
		//
		// Although it's not used later we need to use this file (versus empty or made-up bytes) to enable installation
		// logic to receive data with the correct checksum.
		f, err := os.Open(providerMetadata.Filename)
		if err != nil {
			t.Fatalf("failed to open mock source file: %s", err)
		}
		defer f.Close()
		archiveBytes, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("failed to read mock source file: %s", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(archiveBytes)
	})}

	server.Start()
	t.Cleanup(server.Close)

	return source
}

// newHTTPMirrorProviderSourceUsingTestHttpServer returns an HTTPMirrorSource that is backed
// by a test HTTPS server that acts as a network mirror. The test HTTP server will serve the
// providers and versions defined in the input map using the Provider Network Mirror Protocol.
//
// Calling code has the option of allowing the mirror to report hashes for each provider version,
// or to force the mirror to not report hashes.
//
// All cleanup is handled internally using t.Cleanup.
//
// This source is not sufficient for providers to be available to _use_ during a test,
// it is only sufficient to enable a provider installation process to complete successfully.
// If your test expects a provider to be installed and then used, ensure that testOverrides
// provides the actual provider implementations for use during the test.
func newHTTPMirrorProviderSourceUsingTestHttpServer(t *testing.T, input map[string][]string, allowReturnHashes bool) *getproviders.HTTPMirrorSource {
	t.Helper()

	// Get un-started server so we can obtain the port it'll run on.
	server := httptest.NewUnstartedServer(nil)
	address := server.Listener.Addr().String()

	// Parse input map for convenience
	availableProviderVersions := make(map[addrs.Provider][]getproviders.Version)
	for pSource, versions := range input {
		addr := addrs.MustParseProviderSourceString(pSource)
		var parsedVersions []getproviders.Version
		for _, versionStr := range versions {
			version := getproviders.MustParseVersion(versionStr)
			parsedVersions = append(parsedVersions, version)
		}
		availableProviderVersions[addr] = parsedVersions
	}

	// Create tmp files for each provider version defined in the availableProviderVersions map.
	// These files are later served by the mock network mirror HTTP server.
	tmpFileLocations := map[addrs.Provider]map[getproviders.Version]string{}
	for addr, versions := range availableProviderVersions {
		for _, version := range versions {
			f, _, err := getproviders.CreateFakeFileWithChecksumForProvider(t, addr, version, getproviders.CurrentPlatform, "") // file cleanup already handled
			if err != nil {
				t.Fatalf("failed to create fake package for provider %s %s: %s", addr, version, err)
			}
			if _, ok := tmpFileLocations[addr]; !ok {
				tmpFileLocations[addr] = make(map[getproviders.Version]string)
			}
			tmpFileLocations[addr][version] = f.Name()
		}
	}

	// Implement a network mirror
	//
	// First we define handlers for different endpoints, and then we set up the
	// server to use those handlers.

	// Handler for endpoints that list available versions of a provider
	// GET :hostname/:namespace/:type/index.json
	handleListEndpoint := func(addr addrs.Provider, w http.ResponseWriter, r *http.Request) {
		// Create response body with the versions available for this provider.
		response := getproviders.ListVersionsResponseBody{
			Versions: make(map[string]struct{}),
		}
		versions := availableProviderVersions[addr]
		for _, v := range versions {
			response.Versions[v.String()] = struct{}{}
		}

		b, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal versions response for provider %s: %s", addr, err)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}

	// Handler for endpoints that list data about a specific versions of a provider
	// GET :hostname/:namespace/:type/:version.json
	handleVersionEndpoint := func(addr addrs.Provider, v getproviders.Version, w http.ResponseWriter, r *http.Request) {
		// Create response body with metadata for single package that matches current platform
		response := getproviders.ListInstallationPackagesResponseBody{
			Archives: make(map[string]*getproviders.ListInstallationPackagesArchiveMeta),
		}
		// E.g. terraform-provider-foobar_1.0.0_darwin_amd64.zip
		path := fmt.Sprintf(
			"terraform-provider-%s_%s_%s.zip",
			addr.Type,
			v.String(),
			getproviders.CurrentPlatform.String(),
		)

		// Make hashes
		if _, ok := tmpFileLocations[addr]; !ok {
			t.Fatalf("no package metadata found in the mock network mirror for provider source %q", addr)
		}
		if _, ok := tmpFileLocations[addr][v]; !ok {
			t.Fatalf("no package metadata found in the mock network mirror for provider source %q and version %q", addr, v)
		}
		fileLocation := tmpFileLocations[addr][v]
		zHash, err := getproviders.PackageHashLegacyZipSHA(getproviders.PackageLocalArchive(fileLocation))
		if err != nil {
			t.Fatalf("failed to compute hash for provider source %q and version %q: %s", addr, v, err)
		}
		h1Hash, err := getproviders.PackageHashV1(getproviders.PackageLocalArchive(fileLocation))
		if err != nil {
			t.Fatalf("failed to compute hash for provider source %q and version %q: %s", addr, v, err)
		}

		m := &getproviders.ListInstallationPackagesArchiveMeta{
			RelativeURL: path,
		}
		if allowReturnHashes {
			// Test may want no hashes to be returned, to test the behaviour when no authentication data
			// is available for a provider package.
			m.Hashes = []string{
				zHash.String(),
				h1Hash.String(),
			}
		}
		response.Archives[getproviders.CurrentPlatform.String()] = m

		b, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal versions response for provider %s: %s", addr, err)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}

	// Handler for endpoints that allow downloading a provider archive.
	// GET hostname/:namespace/:type/:filename , where filename always ends in .zip
	handleZipDownloadEndpoint := func(addr addrs.Provider, v getproviders.Version, w http.ResponseWriter, r *http.Request) {
		// This code returns data in the temporary file that's created by this test helper.
		// This 'download' is not used when Terraform uses the provider after the mock installation completes;
		// Terraform will look for will use testOverrides in the Meta set up for this test.
		//
		// Although it's not used later we need to use this file (versus empty or made-up bytes) to enable installation
		// logic to receive data with the correct checksum.
		fileLocation, ok := tmpFileLocations[addr][v]
		if !ok {
			t.Fatalf("no package metadata found in the mock network mirror for provider source %q and version %q", addr, v)
		}

		f, err := os.Open(fileLocation)
		if err != nil {
			t.Fatalf("failed to open mock source file: %s", err)
		}
		defer f.Close()
		archiveBytes, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("failed to read mock source file: %s", err)
		}
		w.Header().Add("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(archiveBytes)
	}

	// Set up how the server handles requests.
	// This includes validating that the request matches a provider version that was specified in the input map
	// from the calling test code.
	server.Config = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			t.Fatalf("unexpected URL path in request to test network mirror: %s", r.URL.Path)
		}

		// Parse provider source; avoid repeated logic in handlers below
		providerSource := strings.Join(parts[1:len(parts)-1], "/")
		providerAddr, diag := addrs.ParseProviderSourceString(providerSource)
		if diag.HasErrors() {
			t.Fatalf("failed to parse provider source from URL path %q: %s", r.URL.Path, diag.Err())
		}

		// Is the request for a provider source and version combo specified by the test?
		if _, ok := availableProviderVersions[providerAddr]; !ok {
			t.Fatalf("provider source %q is not available in the mock network mirror", providerAddr)
		}
		var v getproviders.Version
		if !strings.HasSuffix(r.URL.Path, "/index.json") {
			// Only parse and assert version for requests that include a version in the path
			versionRegex := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)
			versionStr := versionRegex.FindString(r.URL.Path)
			if versionStr == "" {
				t.Fatalf("expected to be able to parse version from URL path %q", r.URL.Path)
			}
			v = getproviders.MustParseVersion(versionStr)

			found := false
			for _, pv := range availableProviderVersions[providerAddr] {
				if pv.Same(v) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("provider source %q and version %q is not available in the mock network mirror", providerAddr, v)
			}
		}

		switch {
		case strings.HasSuffix(r.URL.Path, "/index.json"):
			// List versions for a provider
			// GET :hostname/:namespace/:type/index.json
			handleListEndpoint(providerAddr, w, r)
			return
		case strings.HasSuffix(r.URL.Path, ".json"):
			// Show archives for a specific version
			// GET :hostname/:namespace/:type/:version.json
			handleVersionEndpoint(providerAddr, v, w, r)
			return
		case strings.HasSuffix(r.URL.Path, ".zip"):
			// Handle provider binary download requests.
			// GET :hostname/:namespace/:type/:filename.zip
			handleZipDownloadEndpoint(providerAddr, v, w, r)
			return
		default:
			t.Fatalf("unhandled request to mock network mirror HTTP server:\npath: %s\n request: %#v", r.URL.Path, r)
		}
	})}

	server.Start() // Mock HTTP Mirror source doesn't enforce TLS
	t.Cleanup(server.Close)

	// Set up mock provider source that mocks installation via HTTP.
	url := &url.URL{
		Scheme: "http", // No TLS again
		Host:   address,
	}

	source := getproviders.NewMockHTTPMirrorSource(t, url)

	return source
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
			meta, err := getproviders.FakeInstallablePackageMeta(t, addr, version, getproviders.VersionList{getproviders.MustParseVersion("5.0")}, getproviders.CurrentPlatform, "")
			// We're going to install all these fake packages before we return,
			// so we don't need to preserve them afterwards.
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

func mockSingleStateStoreSchema(storeType string) map[string]providers.Schema {
	return map[string]providers.Schema{
		storeType: {
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
		},
	}
}

func mockPluggableStateStorageProvider(schemas map[string]providers.Schema) *testing_provider.MockProvider {
	// Create a mock provider to use for PSS
	// Get mock provider factory to be used during init
	//
	// This imagines a provider called `test` that contains
	// a pluggable state store implementation called `store`.
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
			StateStores:       schemas,
		},
	}
	return addPluggableStateStoreToMockProvider(&mock, schemas)
}

func addPluggableStateStoreToMockProvider(p *testing_provider.MockProvider, schemas map[string]providers.Schema) *testing_provider.MockProvider {
	p.GetProviderSchemaResponse.StateStores = schemas

	typeNames := slices.Sorted(maps.Keys(schemas))
	p.MockStates = testing_provider.NewMockStateBytesWithTypes(typeNames)
	p.GetStatesFn = func(req providers.GetStatesRequest) (resp providers.GetStatesResponse) {
		stateIds, err := p.MockStates.StateIds(req.TypeName)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
		}
		resp.States = stateIds

		return resp
	}
	p.ConfigureStateStoreFn = func(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
		return providers.ConfigureStateStoreResponse{
			Capabilities: providers.StateStoreServerCapabilities{
				ChunkSize: 1234, // arbitrary number that isn't 0
			},
		}
	}
	p.WriteStateBytesFn = func(req providers.WriteStateBytesRequest) (resp providers.WriteStateBytesResponse) {
		// Workspaces exist once the artefact representing it is written
		err := p.MockStates.Write(req.TypeName, req.StateId, req.Bytes)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
		}

		return resp
	}
	p.ReadStateBytesFn = func(req providers.ReadStateBytesRequest) (resp providers.ReadStateBytesResponse) {
		b, err := p.MockStates.Read(req.TypeName, req.StateId)
		if err != nil {
			if errors.Is(err, testing_provider.StateNotFoundErr{TypeName: req.TypeName, StateId: req.StateId}) {
				warn := tfdiags.SimpleWarning(err.Error())
				resp.Diagnostics = resp.Diagnostics.Append(warn)
			} else {
				resp.Diagnostics = resp.Diagnostics.Append(err)
			}
		}
		resp.Bytes = b

		return resp
	}
	p.DeleteStateFn = func(req providers.DeleteStateRequest) (resp providers.DeleteStateResponse) {
		err := p.MockStates.Delete(req.TypeName, req.StateId)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
		}
		return resp
	}
	return p
}

func assertLockfileContents(t *testing.T, path string, expected string) {
	t.Helper()

	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error accessing lock file: %s", err)
	}
	got := bytes.TrimSpace(buf)
	want := strings.TrimSpace(expected)
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("unexpected difference in lock file (%q) content: %s", path, diff)
	}
}

// If provider installation runs first, discovery of a provider sourced from an aliased hostname
// happens before the alias exists, so it resolves the real hostname (returning
// HTML rather than a discovery document). For the case of "localterraform.com" it'll fail with:
//
//	discovery URL returned an unsupported Content-Type "text/html"
//
// This test asserts the ordering invariant. At the moment provider discovery happens for a provider
// sourced from the aliased hostname, the backend-registered alias must already be present on the shared
// service-discovery object.
func TestInit_backendAliasesRegisteredBeforeProviderDiscovery(t *testing.T) {
	aliasHost, err := svchost.ForComparison("localterraform.com")
	if err != nil {
		t.Fatalf("invalid alias hostname: %s", err)
	}
	targetHost, err := svchost.ForComparison("app.terraform.io")
	if err != nil {
		t.Fatalf("invalid target hostname: %s", err)
	}

	// The shared service-discovery object is given credentials for the alias target
	// If the backend registers the alias before provider discovery
	// (the correct ordering), a credentials lookup for the alias hostname resolves
	// to the target and returns these credentials. If the alias is missing,
	// the lookup for the alias hostname returns no credentials.
	const sentinelToken = "sentinel-token"
	credsSrc := auth.StaticCredentialsSource(map[svchost.Hostname]map[string]any{
		targetHost: {"token": sentinelToken},
	})
	services := disco.NewWithCredentialsSource(credsSrc)

	const backendType = "alias"
	previousBackend := backendInit.Backend(backendType)
	backendInit.Set(backendType, func() backend.Backend {
		return &aliasRegisteringBackend{
			Local: backendLocal.New(),
			alias: backendrun.HostAlias{From: aliasHost, To: targetHost},
		}
	})
	t.Cleanup(func() { backendInit.Set(backendType, previousBackend) })

	// Configuration using the alias-registering backend and requiring a provider
	// sourced from the aliased hostname.
	td := t.TempDir()
	cfg := `terraform {
  backend "alias" {}
  required_providers {
    foo = {
      source = "localterraform.com/foo/bar"
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(cfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Chdir(td)

	watchedProvider := addrs.MustParseProviderSourceString("localterraform.com/foo/bar")
	baseSource := newMockProviderSource(t, map[string][]string{
		"localterraform.com/foo/bar": {"1.0.0"},
	})
	providerSource := &aliasAssertingProviderSource{
		Source:    baseSource,
		t:         t,
		services:  services,
		watch:     watchedProvider,
		aliasHost: aliasHost,
	}

	ui := testUiWrapped(t)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Services:         services,
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	code := c.Run(nil)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("expected successful init, got exit %d:\n%s", code, testOutput.All())
	}

	if !providerSource.checked {
		t.Fatalf("provider discovery for %s never happened; test did not exercise the ordering invariant", watchedProvider)
	}
}

// aliasRegisteringBackend is a backend that advertises a
// service-discovery alias during init, like the cloud/remote backends do.
type aliasRegisteringBackend struct {
	*backendLocal.Local
	alias backendrun.HostAlias
}

func (b *aliasRegisteringBackend) ServiceDiscoveryAliases() ([]backendrun.HostAlias, error) {
	return []backendrun.HostAlias{b.alias}, nil
}

// aliasAssertingProviderSource wraps a provider Source and, the first time it is
// asked for the versions of a watched provider, asserts that a backend-registered
// service-discovery alias for that provider's hostname is already resolvable on
// the shared service-discovery object.
type aliasAssertingProviderSource struct {
	getproviders.Source

	t         *testing.T
	services  *disco.Disco
	watch     addrs.Provider
	aliasHost svchost.Hostname
	checked   bool
}

func (s *aliasAssertingProviderSource) AvailableVersions(ctx context.Context, provider addrs.Provider) (getproviders.VersionList, getproviders.Warnings, error) {
	if provider == s.watch && !s.checked {
		s.checked = true
		creds, err := s.services.CredentialsForHost(s.aliasHost)
		if err != nil {
			s.t.Errorf("unexpected error resolving credentials for %q: %s", s.aliasHost, err)
		}
		if creds == nil {
			s.t.Errorf("backend service-discovery alias for %q was not registered before provider discovery; "+
				"providers must be installed after backend initialization (regression of #38227, fixed by #38648)", s.aliasHost)
		}
	}
	return s.Source.AvailableVersions(ctx, provider)
}

// Test_fetchPackageSuccessCallback asserts how the fetchPackageSuccessCallback function behaves when called with different authentication results.
// This test will be used to check that refactoring how output is made doesn't introduce any unexpected changes.
func Test_fetchPackageSuccessCallback(t *testing.T) {
	const verifiedChecksum = 0
	const officialProvider = 1
	const partnerProvider = 2
	const noKey = ""

	t.Run("no auth result", func(t *testing.T) {
		viewH, doneH := testView(t)
		viewJ, doneJ := testView(t)
		initViewHuman := views.NewInit(arguments.ViewHuman, viewH)
		initViewJSON := views.NewInit(arguments.ViewJSON, viewJ)

		cbHuman := fetchPackageSuccessCallback(initViewHuman)
		cbJSON := fetchPackageSuccessCallback(initViewJSON)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		localDir := "."

		var authResult *getproviders.PackageAuthenticationResult = nil

		// Use callback to log output
		cbHuman(p, ver, localDir, authResult)
		cbJSON(p, ver, localDir, authResult)

		// Assert output - human
		output := doneH(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (unauthenticated)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
		// Assert output - json
		output = doneJ(t)
		expectedOutput = `{"@level":"info","@message":"Installed provider version: hashicorp/test v1.2.3 (unauthenticated)","@module":"terraform.ui","@timestamp":` // Stop comparison before timestamp
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result", func(t *testing.T) {
		viewH, doneH := testView(t)
		viewJ, doneJ := testView(t)
		initViewHuman := views.NewInit(arguments.ViewHuman, viewH)
		initViewJSON := views.NewInit(arguments.ViewJSON, viewJ)

		cbHuman := fetchPackageSuccessCallback(initViewHuman)
		cbJSON := fetchPackageSuccessCallback(initViewJSON)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		localDir := "."

		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		// Use callback to log output
		cbHuman(p, ver, localDir, authResult)
		cbJSON(p, ver, localDir, authResult)

		// Assert output - human
		output := doneH(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (verified checksum)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
		// Assert output - json
		output = doneJ(t)
		expectedOutput = `{"@level":"info","@message":"Installed provider version: hashicorp/test v1.2.3 (verified checksum)","@module":"terraform.ui","@timestamp":` // Stop comparison before timestamp
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("official provider auth result", func(t *testing.T) {
		viewH, doneH := testView(t)
		viewJ, doneJ := testView(t)
		initViewHuman := views.NewInit(arguments.ViewHuman, viewH)
		initViewJSON := views.NewInit(arguments.ViewJSON, viewJ)

		cbHuman := fetchPackageSuccessCallback(initViewHuman)
		cbJSON := fetchPackageSuccessCallback(initViewJSON)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		localDir := "."

		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, noKey)

		// Use callback to log output
		cbHuman(p, ver, localDir, authResult)
		cbJSON(p, ver, localDir, authResult)

		// Assert output - human
		output := doneH(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by HashiCorp)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
		// Assert output - json
		output = doneJ(t)
		expectedOutput = `{"@level":"info","@message":"Installed provider version: hashicorp/test v1.2.3 (signed by HashiCorp)","@module":"terraform.ui","@timestamp":` // Stop comparison before timestamp
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("third party signed partner provider with key id", func(t *testing.T) {
		viewH, doneH := testView(t)
		viewJ, doneJ := testView(t)
		initViewHuman := views.NewInit(arguments.ViewHuman, viewH)
		initViewJSON := views.NewInit(arguments.ViewJSON, viewJ)

		cbHuman := fetchPackageSuccessCallback(initViewHuman)
		cbJSON := fetchPackageSuccessCallback(initViewJSON)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		localDir := "."

		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		// Use callback to log output
		cbHuman(p, ver, localDir, authResult)
		cbJSON(p, ver, localDir, authResult)

		// Assert output - human
		output := doneH(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by a HashiCorp partner, key ID key-id-123)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
		// Assert output - json
		output = doneJ(t)
		expectedOutput = `{"@level":"info","@message":"Installed provider version: hashicorp/test v1.2.3 (signed by a HashiCorp partnerkey_id: key-id-123)","@module":"terraform.ui","@timestamp":` // Stop comparison before timestamp
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}
