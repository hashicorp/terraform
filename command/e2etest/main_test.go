package e2etest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/e2e"
)

var terraformBin string

func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() func() {
	if terraformBin != "" {
		// this is pre-set when we're running in a binary produced from
		// the make-archive.sh script, since that builds a ready-to-go
		// binary into the archive. However, we do need to turn it into
		// an absolute path so that we can find it when we change the
		// working directory during tests.
		var err error
		terraformBin, err = filepath.Abs(terraformBin)
		if err != nil {
			panic(fmt.Sprintf("failed to find absolute path of terraform executable: %s", err))
		}
		return func() {}
	}

	tmpFilename := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

	// Make the executable available for use in tests
	terraformBin = tmpFilename

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
