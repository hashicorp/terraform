package e2etest

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

// TestProviderDevOverrides is a test that terraform can execute a 3rd party
// provisioner plugin.
func TestProvisioner(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	tf := e2e.NewBinary(terraformBin, "testdata/provisioner")
	defer tf.Close()

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
