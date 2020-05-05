package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/mitchellh/cli"
)

func TestZeroThirteenUpgrade_success(t *testing.T) {
	testCases := map[string]struct {
		path string
		args []string
		out  string
	}{
		"implicit": {
			path: "013upgrade-implicit-providers",
			out:  "providers.tf",
		},
		"explicit": {
			path: "013upgrade-explicit-providers",
			out:  "providers.tf",
		},
		"subdir": {
			path: "013upgrade-subdir",
			args: []string{"subdir"},
			out:  "subdir/providers.tf",
		},
		"fileExists": {
			path: "013upgrade-file-exists",
			out:  "providers-1.tf",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			td := tempDir(t)
			copy.CopyDir(testFixturePath(tc.path), td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			ui := new(cli.MockUi)
			c := &ZeroThirteenUpgradeCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
			}

			if code := c.Run(tc.args); code != 0 {
				t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
			}

			output := ui.OutputWriter.String()
			if !strings.Contains(output, "Upgrade complete") {
				t.Fatal("unexpected output:", output)
			}

			actual, err := ioutil.ReadFile(tc.out)
			if err != nil {
				t.Fatalf("failed to read output %s: %s", tc.out, err)
			}
			expected, err := ioutil.ReadFile("expected/providers.tf")
			if err != nil {
				t.Fatal("failed to read expected/providers.tf", err)
			}

			if !bytes.Equal(actual, expected) {
				t.Fatalf("actual output: \n%s\nexpected output: \n%s", string(actual), string(expected))
			}
		})
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

	if code := c.Run(nil); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "Not a module directory") {
		t.Fatal("unexpected error:", errMsg)
	}
}

func TestZeroThirteenUpgrade_invalidProviderVersion(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("013upgrade-invalid"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ZeroThirteenUpgradeCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run(nil); code == 0 {
		t.Fatal("expected error, got:", ui.OutputWriter)
	}

	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "Invalid provider version constraint") {
		t.Fatal("unexpected error:", errMsg)
	}
}

func TestZeroThirteenUpgrade_noProviders(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("013upgrade-no-providers"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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
	if !strings.Contains(output, "No non-default providers found") {
		t.Fatal("unexpected output:", output)
	}

	if _, err := os.Stat("providers.tf"); !os.IsNotExist(err) {
		t.Fatal("unexpected providers.tf created")
	}
}
