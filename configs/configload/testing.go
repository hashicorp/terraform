package configload

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// NewLoaderForTests is a variant of NewLoader that is intended to be more
// convenient for unit tests.
//
// The loader's modules directory is a separate temporary directory created
// for each call. Along with the created loader, this function returns a
// cleanup function that should be called before the test completes in order
// to remove that temporary directory.
//
// In the case of any errors, t.Fatal (or similar) will be called to halt
// execution of the test, so the calling test does not need to handle errors
// itself.
func NewLoaderForTests(t *testing.T) (*Loader, func()) {
	t.Helper()

	modulesDir, err := ioutil.TempDir("", "tf-configs")
	if err != nil {
		t.Fatalf("failed to create temporary modules dir: %s", err)
		return nil, func() {}
	}

	cleanup := func() {
		os.RemoveAll(modulesDir)
	}

	loader, err := NewLoader(&Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		cleanup()
		t.Fatalf("failed to create config loader: %s", err)
		return nil, func() {}
	}

	return loader, cleanup
}

// LoadConfigForTests is a convenience wrapper around NewLoaderForTests,
// Loader.InstallModules and Loader.LoadConfig that allows a test configuration
// to be loaded in a single step.
//
// If module installation fails, t.Fatal (or similar) is called to halt
// execution of the test, under the assumption that installation failures are
// not expected. If installation failures _are_ expected then use
// NewLoaderForTests and work with the loader object directly. If module
// installation succeeds but generates warnings, these warnings are discarded.
//
// If installation succeeds but errors are detected during loading then a
// possibly-incomplete config is returned along with error diagnostics. The
// test run is not aborted in this case, so that the caller can make assertions
// against the returned diagnostics.
//
// As with NewLoaderForTests, a cleanup function is returned which must be
// called before the test completes in order to remove the temporary
// modules directory.
func LoadConfigForTests(t *testing.T, rootDir string) (*configs.Config, *Loader, func(), tfdiags.Diagnostics) {
	t.Helper()

	var diags tfdiags.Diagnostics

	loader, cleanup := NewLoaderForTests(t)
	hclDiags := loader.InstallModules(rootDir, true, InstallHooksImpl{})
	if diags.HasErrors() {
		cleanup()
		diags = diags.Append(hclDiags)
		t.Fatal(diags.Err())
		return nil, nil, cleanup, diags
	}

	config, hclDiags := loader.LoadConfig(rootDir)
	diags = diags.Append(hclDiags)
	return config, loader, cleanup, diags
}

// MustLoadConfigForTests is a variant of LoadConfigForTests which calls
// t.Fatal (or similar) if there are any errors during loading, and thus
// does not return diagnostics at all.
//
// This is useful for concisely writing tests that don't expect errors at
// all. For tests that expect errors and need to assert against them, use
// LoadConfigForTests instead.
func MustLoadConfigForTests(t *testing.T, rootDir string) (*configs.Config, *Loader, func()) {
	t.Helper()

	config, loader, cleanup, diags := LoadConfigForTests(t, rootDir)
	if diags.HasErrors() {
		cleanup()
		t.Fatal(diags.Err())
	}
	return config, loader, cleanup
}
