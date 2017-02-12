package experiment

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// Test experiments
var (
	X_test1 = newBasicID("test1", "TEST1", false)
	X_test2 = newBasicID("test2", "TEST2", true)
)

// Reinitializes the package to a clean slate
func testReinit() {
	All = []ID{X_test1, X_test2, x_force}
	reload()
}

func init() {
	testReinit()

	// Clear all env vars so they don't affect tests
	for _, id := range All {
		os.Unsetenv(fmt.Sprintf("TF_X_%s", id.Env()))
	}
}

func TestDefault(t *testing.T) {
	testReinit()

	if Enabled(X_test1) {
		t.Fatal("test1 should not be enabled")
	}

	if !Enabled(X_test2) {
		t.Fatal("test2 should be enabled")
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("TF_X_TEST2", "0")
	defer os.Unsetenv("TF_X_TEST2")

	testReinit()

	if Enabled(X_test2) {
		t.Fatal("test2 should be enabled")
	}
}

func TestFlag(t *testing.T) {
	testReinit()

	// Verify default
	if !Enabled(X_test2) {
		t.Fatal("test2 should be enabled")
	}

	// Setup a flag set
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	Flag(fs)
	fs.Parse([]string{"-Xtest2=false"})

	if Enabled(X_test2) {
		t.Fatal("test2 should not be enabled")
	}
}

func TestFlag_overEnv(t *testing.T) {
	os.Setenv("TF_X_TEST2", "1")
	defer os.Unsetenv("TF_X_TEST2")

	testReinit()

	// Verify default
	if !Enabled(X_test2) {
		t.Fatal("test2 should be enabled")
	}

	// Setup a flag set
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	Flag(fs)
	fs.Parse([]string{"-Xtest2=false"})

	if Enabled(X_test2) {
		t.Fatal("test2 should not be enabled")
	}
}

func TestForce(t *testing.T) {
	os.Setenv("TF_X_FORCE", "1")
	defer os.Unsetenv("TF_X_FORCE")

	testReinit()

	if !Force() {
		t.Fatal("should force")
	}
}

func TestForce_flag(t *testing.T) {
	os.Unsetenv("TF_X_FORCE")

	testReinit()

	// Setup a flag set
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	Flag(fs)
	fs.Parse([]string{"-Xforce"})

	if !Force() {
		t.Fatal("should force")
	}
}
