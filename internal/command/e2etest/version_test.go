// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/version"
)

func TestVersion(t *testing.T) {
	// Along with testing the "version" command in particular, this serves
	// as a good smoke test for whether the Terraform binary can even be
	// compiled and run, since it doesn't require any external network access
	// to do its job.

	t.Parallel()

	fixturePath := filepath.Join("testdata", "empty")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	stdout, stderr, err := tf.Run("version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	wantVersion := fmt.Sprintf("Terraform v%s", version.String())
	if !strings.Contains(stdout, wantVersion) {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}
}

func TestVersionWithProvider(t *testing.T) {
	// This is a more elaborate use of "version" that shows the selected
	// versions of plugins too.
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "template-provider")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	configFile := emptyConfigFileForTests(t, tf.Path(""))
	tf.AddEnv(fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", configFile))

	// Initial run (before "init") should work without error but will not
	// include the provider version, since we've not "locked" one yet.
	{
		stdout, stderr, err := tf.Run("version")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		wantVersion := fmt.Sprintf("Terraform v%s", version.String())
		if !strings.Contains(stdout, wantVersion) {
			t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
		}
	}

	{
		_, _, err := tf.Run("init")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}

	// After running init, we additionally include information about the
	// selected version of the "template" provider.
	{
		stdout, stderr, err := tf.Run("version")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		wantMsg := "+ provider registry.terraform.io/hashicorp/template v" // we don't know which version we'll get here
		if !strings.Contains(stdout, wantMsg) {
			t.Errorf("output does not contain provider information %q:\n%s", wantMsg, stdout)
		}
	}
}

// If users run any command with a -version or -v flag, we reroute to the version command.
// This test ensures that this rerouting works as expected and defines how additional flags and arguments are handled.
func TestVersionReroutingFromOtherCommands(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "full-workflow-null")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	wantVersion := fmt.Sprintf("Terraform v%s\non %s\n\n", version.String(), getproviders.CurrentPlatform.String())

	// Use version command directly
	// The version command receives no arguments.
	stdout, stderr, err := tf.Run("version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}
	if stdout != wantVersion {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}

	// Use version flag with no command
	// The version command receives arguments: ["-version"]
	// and accepts the flag.
	stdout, stderr, err = tf.Run("-version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}
	if stdout != wantVersion {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}

	// Get version via init command
	// The version command receives arguments: ["init", "-version"]
	// but ignores them all.
	stdout, stderr, err = tf.Run("init", "-version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}
	if stdout != wantVersion {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}

	// Get version via init command with additional global and init-specific flags present
	// The version command receives arguments: ["init", "-version", "-input=false", "-no-color", "-get=false", "-upgrade"]
	// but ignores them all.
	stdout, stderr, err = tf.Run("init", "-version", "-input=false", "-no-color", "-get=false", "-upgrade")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}
	if stdout != wantVersion {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}
}
