package e2etest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

var terraformBin string

// canRunGoBuild is a short-term compromise to account for the fact that we
// have a small number of tests that work by building helper programs using
// "go build" at runtime, but we can't do that in our isolated test mode
// driven by the make-archive.sh script.
//
// FIXME: Rework this a bit so that we build the necessary helper programs
// (test plugins, etc) as part of the initial suite setup, and in the
// make-archive.sh script, so that we can run all of the tests in both
// situations with the tests just using the executable already built for
// them, as we do for terraformBin.
var canRunGoBuild bool

func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() func() {
	if terraformBin != "" {
		// this is pre-set when we're running in a binary produced from
		// the make-archive.sh script, since that is for testing an
		// executable obtained from a real release package. However, we do
		// need to turn it into an absolute path so that we can find it
		// when we change the working directory during tests.
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

	// Tests running in the ad-hoc testing mode are allowed to use "go build"
	// and similar to produce other test executables.
	// (See the comment on this variable's declaration for more information.)
	canRunGoBuild = true

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
	t.Helper()

	if !canAccessNetwork() {
		t.Skip("network access not allowed; use TF_ACC=1 to enable")
	}
}
