package e2etest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

// TestProviderDevOverrides is a test for the special dev_overrides setting
// in the provider_installation section of the CLI configuration file, which
// is our current answer to smoothing provider development by allowing
// developers to opt out of the version number and checksum verification
// we normally do, so they can just overwrite the same local executable
// in-place to iterate faster.
func TestProviderDevOverrides(t *testing.T) {
	t.Parallel()

	tf := e2e.NewBinary(terraformBin, "testdata/provider-dev-override")
	defer tf.Close()

	// In order to do a decent end-to-end test for this case we will need a
	// real enough provider plugin to try to run and make sure we are able
	// to actually run it. For now we'll use the "test" provider for that,
	// because it happens to be in this repository and therefore allows
	// us to avoid drawing in anything external, but we might revisit this
	// strategy in future if other needs cause us to evolve the test
	// provider in a way that makes it less suitable for this particular test,
	// such as if it stops being buildable into an independent executable.
	providerExeDir := filepath.Join(tf.WorkDir(), "pkgdir")
	providerExePrefix := filepath.Join(providerExeDir, "terraform-provider-test_")
	providerExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", providerExePrefix)
	t.Logf("temporary provider executable is %s", providerExe)

	err := ioutil.WriteFile(filepath.Join(tf.WorkDir(), "dev.tfrc"), []byte(fmt.Sprintf(`
		provider_installation {
			dev_overrides {
				"example.com/test/test" = %q
			}
		}
	`, providerExeDir)), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	tf.AddEnv("TF_CLI_CONFIG_FILE=dev.tfrc")

	stdout, stderr, err := tf.Run("providers")
	if err != nil {
		t.Fatalf("unexpected error: %s\n%s", err, stderr)
	}
	if got, want := stdout, `provider[example.com/test/test]`; !strings.Contains(got, want) {
		t.Errorf("configuration should depend on %s, but doesn't\n%s", want, got)
	}

	// NOTE: We're intentionally not running "terraform init" here, because
	// dev overrides are always ready to use and don't need any special action
	// to "install" them. This test is mimicking the a happy path of going
	// directly from "go build" to validate/plan/apply without interacting
	// with any registries, mirrors, lock files, etc. To verify "terraform
	// init" does actually show a warning, that behavior is tested at the end.
	stdout, stderr, err = tf.Run("validate")
	if err != nil {
		t.Fatalf("unexpected error: %s\n%s", err, stderr)
	}

	if got, want := stdout, `The configuration is valid, but`; !strings.Contains(got, want) {
		t.Errorf("stdout doesn't include the success message\nwant: %s\n%s", want, got)
	}
	if got, want := stdout, `Provider development overrides are in effect`; !strings.Contains(got, want) {
		t.Errorf("stdout doesn't include the warning about development overrides\nwant: %s\n%s", want, got)
	}

	stdout, stderr, err = tf.Run("init")
	if err == nil {
		t.Fatal("expected error: Failed to query available provider packages")
	}
	if got, want := stdout, `Provider development overrides are in effect`; !strings.Contains(got, want) {
		t.Errorf("stdout doesn't include the warning about development overrides\nwant: %s\n%s", want, got)
	}
	if got, want := stderr, `Failed to query available provider packages`; !strings.Contains(got, want) {
		t.Errorf("stderr doesn't include the error about listing unavailable development provider\nwant: %s\n%s", want, got)
	}
}
