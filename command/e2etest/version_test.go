package e2etest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/e2e"
	tfcore "github.com/hashicorp/terraform/terraform"
)

func TestVersion(t *testing.T) {
	// Along with testing the "version" command in particular, this serves
	// as a good smoke test for whether the Terraform binary can even be
	// compiled and run, since it doesn't require any external network access
	// to do its job.

	t.Parallel()

	fixturePath := filepath.Join("test-fixtures", "empty")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, stderr, err := tf.Run("version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	wantVersion := fmt.Sprintf("Terraform v%s", tfcore.VersionString())
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

	fixturePath := filepath.Join("test-fixtures", "template-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

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

		wantVersion := fmt.Sprintf("Terraform v%s", tfcore.VersionString())
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

		wantMsg := "+ provider.template v" // we don't know which version we'll get here
		if !strings.Contains(stdout, wantMsg) {
			t.Errorf("output does not contain provider information %q:\n%s", wantMsg, stdout)
		}
	}
}
