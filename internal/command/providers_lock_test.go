package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestProvidersLock(t *testing.T) {
	t.Run("noop", func(t *testing.T) {
		// in the most basic case, running providers lock in a directory with no configuration at all should succeed.
		// create an empty working directory
		td := tempDir(t)
		os.MkdirAll(td, 0755)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		ui := new(cli.MockUi)
		c := &ProvidersLockCommand{
			Meta: Meta{
				Ui: ui,
			},
		}
		code := c.Run([]string{})
		if code != 0 {
			t.Fatalf("wrong exit code; expected 0, got %d", code)
		}
	})

	// This test depends on the -fs-mirror argument, so we always know what results to expect
	t.Run("basic", func(t *testing.T) {
		td := tempDir(t)
		testCopyDir(t, testFixturePath("providers-lock/basic"), td)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		// Our fixture dir has a generic os_arch dir, which we need to customize
		// to the actual OS/arch where this test is running in order to get the
		// desired result.
		fixtMachineDir := filepath.Join(td, "fs-mirror/registry.terraform.io/hashicorp/test/1.0.0/os_arch")
		wantMachineDir := filepath.Join(td, "fs-mirror/registry.terraform.io/hashicorp/test/1.0.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
		err := os.Rename(fixtMachineDir, wantMachineDir)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		p := testProvider()
		ui := new(cli.MockUi)
		c := &ProvidersLockCommand{
			Meta: Meta{
				Ui:               ui,
				testingOverrides: metaOverridesForProvider(p),
			},
		}

		args := []string{"-fs-mirror=fs-mirror"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit code; expected 0, got %d", code)
		}

		lockfile, err := os.ReadFile(".terraform.lock.hcl")
		if err != nil {
			t.Fatal("error reading lockfile")
		}

		expected := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/test" {
  version = "1.0.0"
  hashes = [
    "h1:7MjN4eFisdTv4tlhXH5hL4QQd39Jy4baPhFxwAd/EFE=",
  ]
}
`
		if string(lockfile) != expected {
			t.Fatalf("wrong lockfile content")
		}
	})
}

func TestProvidersLock_args(t *testing.T) {

	t.Run("mirror collision", func(t *testing.T) {
		ui := new(cli.MockUi)
		c := &ProvidersLockCommand{
			Meta: Meta{
				Ui: ui,
			},
		}

		// only one of these arguments can be used at a time
		args := []string{
			"-fs-mirror=/foo/",
			"-net-mirror=www.foo.com",
		}
		code := c.Run(args)

		if code != 1 {
			t.Fatalf("wrong exit code; expected 1, got %d", code)
		}
		output := ui.ErrorWriter.String()
		if !strings.Contains(output, "The -fs-mirror and -net-mirror command line options are mutually-exclusive.") {
			t.Fatalf("missing expected error message: %s", output)
		}
	})

	t.Run("invalid platform", func(t *testing.T) {
		ui := new(cli.MockUi)
		c := &ProvidersLockCommand{
			Meta: Meta{
				Ui: ui,
			},
		}

		// not a valid platform
		args := []string{"-platform=arbitrary_nonsense_that_isnt_valid"}
		code := c.Run(args)

		if code != 1 {
			t.Fatalf("wrong exit code; expected 1, got %d", code)
		}
		output := ui.ErrorWriter.String()
		if !strings.Contains(output, "must be two words separated by an underscore.") {
			t.Fatalf("missing expected error message: %s", output)
		}
	})

	t.Run("invalid provider argument", func(t *testing.T) {
		ui := new(cli.MockUi)
		c := &ProvidersLockCommand{
			Meta: Meta{
				Ui: ui,
			},
		}

		// There is no configuration, so it's not valid to use any provider argument
		args := []string{"hashicorp/random"}
		code := c.Run(args)

		if code != 1 {
			t.Fatalf("wrong exit code; expected 1, got %d", code)
		}
		output := ui.ErrorWriter.String()
		if !strings.Contains(output, "The provider registry.terraform.io/hashicorp/random is not required by the\ncurrent configuration.") {
			t.Fatalf("missing expected error message: %s", output)
		}
	})
}
