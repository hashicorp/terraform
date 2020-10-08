package configload

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// tempChdir copies the contents of the given directory to a temporary
// directory and changes the test process's current working directory to
// point to that directory. Also returned is a function that should be
// called at the end of the test (e.g. via "defer") to restore the previous
// working directory.
//
// Tests using this helper cannot safely be run in parallel with other tests.
func tempChdir(t *testing.T, sourceDir string) (string, func()) {
	t.Helper()

	tmpDir, err := ioutil.TempDir("", "terraform-configload")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
		return "", nil
	}

	if err := copyDir(tmpDir, sourceDir); err != nil {
		t.Fatalf("failed to copy fixture to temporary directory: %s", err)
		return "", nil
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to determine current working directory: %s", err)
		return "", nil
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to switch to temp dir %s: %s", tmpDir, err)
		return "", nil
	}

	t.Logf("tempChdir switched to %s after copying from %s", tmpDir, sourceDir)

	return tmpDir, func() {
		err := os.Chdir(oldDir)
		if err != nil {
			panic(fmt.Errorf("failed to restore previous working directory %s: %s", oldDir, err))
		}

		if os.Getenv("TF_CONFIGLOAD_TEST_KEEP_TMP") == "" {
			os.RemoveAll(tmpDir)
		}
	}
}

// tempChdirLoader is a wrapper around tempChdir that also returns a Loader
// whose modules directory is at the conventional location within the
// created temporary directory.
func tempChdirLoader(t *testing.T, sourceDir string) (*Loader, func()) {
	t.Helper()

	_, done := tempChdir(t, sourceDir)
	modulesDir := filepath.Clean(".terraform/modules")

	err := os.MkdirAll(modulesDir, os.ModePerm)
	if err != nil {
		done() // undo the chdir in tempChdir so we can safely run other tests
		t.Fatalf("failed to create modules directory: %s", err)
		return nil, nil
	}

	loader, err := NewLoader(&Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		done() // undo the chdir in tempChdir so we can safely run other tests
		t.Fatalf("failed to create loader: %s", err)
		return nil, nil
	}

	return loader, done
}

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}

func assertDiagnosticSummary(t *testing.T, diags hcl.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %s", diag)
	}
	return true
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if diff := deep.Equal(got, want); diff != nil {
		for _, problem := range diff {
			t.Errorf("%s", problem)
		}
		return true
	}
	return false
}

func assertResultCtyEqual(t *testing.T, got, want cty.Value) bool {
	t.Helper()
	if !got.RawEquals(want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		return true
	}
	return false
}
