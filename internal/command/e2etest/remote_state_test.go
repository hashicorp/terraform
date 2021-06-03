package e2etest

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestTerraformProviderRead(t *testing.T) {
	// Ensure the terraform provider can correctly read a remote state

	t.Parallel()
	fixturePath := filepath.Join("testdata", "terraform-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	//// INIT
	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}
}
