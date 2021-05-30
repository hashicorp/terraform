package e2etest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

// TestProvisionerPlugin is a test that terraform can execute a 3rd party
// provisioner plugin.
func TestProvisionerPlugin(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	tf := e2e.NewBinary(terraformBin, "testdata/provisioner-plugin")
	defer tf.Close()

	// In order to do a decent end-to-end test for this case we will need a
	// real enough provisioner plugin to try to run and make sure we are able
	// to actually run it. Here will build the local-exec provisioner into a
	// binary called test-provisioner
	provisionerExePrefix := filepath.Join(tf.WorkDir(), "terraform-provisioner-test_")
	provisionerExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provisioner-local-exec/main", provisionerExePrefix)

	// provisioners must use the old binary name format, so rename this binary
	newExe := filepath.Join(tf.WorkDir(), "terraform-provisioner-test")
	if _, err := os.Stat(newExe); !os.IsNotExist(err) {
		t.Fatalf("%q already exists", newExe)
	}
	if err := os.Rename(provisionerExe, newExe); err != nil {
		t.Fatalf("error renaming provisioner binary: %v", err)
	}
	provisionerExe = newExe

	t.Logf("temporary provisioner executable is %s", provisionerExe)

	//// INIT
	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY
	stdout, stderr, err := tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "HelloProvisioner") {
		t.Fatalf("missing provisioner output:\n%s", stdout)
	}
}
