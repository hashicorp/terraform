package command

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/mitchellh/cli"
)

func verifyExpectedFiles(t *testing.T, expectedPath string) {
	// Compare output and expected file trees
	var outputFiles, expectedFiles []string

	// Gather list of output files in the current working directory
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			outputFiles = append(outputFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal("error listing output files:", err)
	}

	// Gather list of expected files
	revertChdir := testChdir(t, expectedPath)
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			expectedFiles = append(expectedFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal("error listing expected files:", err)
	}
	revertChdir()

	// If the file trees don't match, give up early
	if diff := cmp.Diff(expectedFiles, outputFiles); diff != "" {
		t.Fatalf("expected and output file trees do not match\n%s", diff)
	}

	// Check that the contents of each file is correct
	for _, filePath := range outputFiles {
		output, err := ioutil.ReadFile(path.Join(".", filePath))
		if err != nil {
			t.Fatalf("failed to read output %s: %s", filePath, err)
		}
		expected, err := ioutil.ReadFile(path.Join(expectedPath, filePath))
		if err != nil {
			t.Fatalf("failed to read expected %s: %s", filePath, err)
		}

		if diff := cmp.Diff(string(expected), string(output)); diff != "" {
			t.Fatalf("expected and output file for %s do not match\n%s", filePath, diff)
		}
	}
}

func TestZeroThirteenUpgrade_success(t *testing.T) {
	registrySource, close := testRegistrySource(t)
	defer close()

	testCases := map[string]string{
		"implicit":              "013upgrade-implicit-providers",
		"explicit":              "013upgrade-explicit-providers",
		"provider not found":    "013upgrade-provider-not-found",
		"implicit not found":    "013upgrade-implicit-not-found",
		"file exists":           "013upgrade-file-exists",
		"no providers":          "013upgrade-no-providers",
		"submodule":             "013upgrade-submodule",
		"providers with source": "013upgrade-providers-with-source",
		"preserves comments":    "013upgrade-preserves-comments",
		"multiple blocks":       "013upgrade-multiple-blocks",
		"multiple files":        "013upgrade-multiple-files",
		"existing versions.tf":  "013upgrade-existing-versions-tf",
		"skipped files":         "013upgrade-skipped-files",
		"provider redirect":     "013upgrade-provider-redirect",
		"version unavailable":   "013upgrade-provider-redirect-version-unavailable",
	}
	for name, testPath := range testCases {
		t.Run(name, func(t *testing.T) {
			inputPath, err := filepath.Abs(testFixturePath(path.Join(testPath, "input")))
			if err != nil {
				t.Fatalf("failed to find input path %s: %s", testPath, err)
			}

			expectedPath, err := filepath.Abs(testFixturePath(path.Join(testPath, "expected")))
			if err != nil {
				t.Fatalf("failed to find expected path %s: %s", testPath, err)
			}

			td := tempDir(t)
			copy.CopyDir(inputPath, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			ui := new(cli.MockUi)
			c := &ZeroThirteenUpgradeCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					ProviderSource:   registrySource,
					Ui:               ui,
				},
			}

			if code := c.Run([]string{"-yes"}); code != 0 {
				t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
			}

			output := ui.OutputWriter.String()
			if !strings.Contains(output, "Upgrade complete") {
				t.Fatal("unexpected output:", output)
			}

			verifyExpectedFiles(t, expectedPath)
		})
	}
}

// Ensure that non-default upgrade paths are supported, and that the output is
// in the correct place. This test is very similar to the table tests above,
// but with a different expected output path, and with an argument passed to
// the Run call.
func TestZeroThirteenUpgrade_submodule(t *testing.T) {
	registrySource, close := testRegistrySource(t)
	defer close()

	testPath := "013upgrade-submodule"

	inputPath, err := filepath.Abs(testFixturePath(path.Join(testPath, "input")))
	if err != nil {
		t.Fatalf("failed to find input path %s: %s", testPath, err)
	}

	// The expected output for processing a submodule is different
	expectedPath, err := filepath.Abs(testFixturePath(path.Join(testPath, "expected-module")))
	if err != nil {
		t.Fatalf("failed to find expected path %s: %s", testPath, err)
	}

	td := tempDir(t)
	copy.CopyDir(inputPath, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			ProviderSource:   registrySource,
			Ui:               ui,
		},
	}

	// Here we pass a target module directory to process
	if code := c.Run([]string{"-yes", "module"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Upgrade complete") {
		t.Fatal("unexpected output:", output)
	}

	verifyExpectedFiles(t, expectedPath)
}

// Verify that JSON and override files are skipped with a warning. Generated
// output for this config is verified in the table driven tests above.
func TestZeroThirteenUpgrade_skippedFiles(t *testing.T) {
	inputPath := testFixturePath(path.Join("013upgrade-skipped-files", "input"))

	td := tempDir(t)
	copy.CopyDir(inputPath, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{"-yes"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Upgrade complete") {
		t.Fatal("unexpected output:", output)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, `The JSON configuration file "variables.tf.json" was skipped`) {
		t.Fatal("missing JSON skipped file warning:", errMsg)
	}
	if !strings.Contains(errMsg, `The override configuration file "bar_override.tf" was skipped`) {
		t.Fatal("missing override skipped file warning:", errMsg)
	}
}

func TestZeroThirteenUpgrade_confirm(t *testing.T) {
	inputPath := testFixturePath(path.Join("013upgrade-explicit-providers", "input"))

	td := tempDir(t)
	copy.CopyDir(inputPath, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"yes"})()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run(nil); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Upgrade complete") {
		t.Fatal("unexpected output:", output)
	}
}

func TestZeroThirteenUpgrade_cancel(t *testing.T) {
	inputPath := testFixturePath(path.Join("013upgrade-explicit-providers", "input"))

	td := tempDir(t)
	copy.CopyDir(inputPath, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Ask input
	defer testInteractiveInput(t, []string{"no"})()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run(nil); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.Contains(output, "Upgrade cancelled") {
		t.Fatal("unexpected output:", output)
	}
	if strings.Contains(output, "Upgrade complete") {
		t.Fatal("unexpected output:", output)
	}
}

func TestZeroThirteenUpgrade_unsupportedVersion(t *testing.T) {
	inputPath := testFixturePath("013upgrade-unsupported-version")

	td := tempDir(t)
	copy.CopyDir(inputPath, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{"-yes"}); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, `Unsupported Terraform Core version`) {
		t.Fatal("missing version constraint error:", errMsg)
	}
}

func TestZeroThirteenUpgrade_invalidFlags(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{"--whoops"}); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "Usage: terraform 0.13upgrade") {
		t.Fatal("unexpected error:", errMsg)
	}
}

func TestZeroThirteenUpgrade_tooManyArguments(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{".", "./modules/test"}); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "Error: Too many arguments") {
		t.Fatal("unexpected error:", errMsg)
	}
}

func TestZeroThirteenUpgrade_empty(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run([]string{"-yes"}); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "Not a module directory") {
		t.Fatal("unexpected error:", errMsg)
	}
}
