package e2etest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/e2e"
)

var bundleBin string

func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() func() {
	tmpFilename := e2e.GoBuild("github.com/hashicorp/terraform/tools/terraform-bundle", "terraform-bundle")
	bundleBin = tmpFilename

	return func() {
		os.Remove(tmpFilename)
	}
}

func canAccessNetwork() bool {
	// We re-use the flag normally used for acceptance tests since that's
	// established as a way to opt-in to reaching out to real systems that
	// may suffer transient errors.
	return os.Getenv("TF_ACC") != ""
}

func skipIfCannotAccessNetwork(t *testing.T) {
	if !canAccessNetwork() {
		t.Skip("network access not allowed; use TF_ACC=1 to enable")
	}
}
