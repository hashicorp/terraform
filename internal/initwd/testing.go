package initwd

import (
	"github.com/hashicorp/terraform/registry"
	"testing"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/tfdiags"
)

// LoadConfigForTests is a convenience wrapper around configload.NewLoaderForTests,
// ModuleInstaller.InstallModules and configload.Loader.LoadConfig that allows
// a test configuration to be loaded in a single step.
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
func LoadConfigForTests(t *testing.T, rootDir string) (*configs.Config, *configload.Loader, func(), tfdiags.Diagnostics) {
	t.Helper()

	var diags tfdiags.Diagnostics

	loader, cleanup := configload.NewLoaderForTests(t)
	inst := NewModuleInstaller(loader.ModulesDir(), registry.NewClient(nil, nil))

	moreDiags := inst.InstallModules(rootDir, true, ModuleInstallHooksImpl{})
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		cleanup()
		t.Fatal(diags.Err())
		return nil, nil, func() {}, diags
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
func MustLoadConfigForTests(t *testing.T, rootDir string) (*configs.Config, *configload.Loader, func()) {
	t.Helper()

	config, loader, cleanup, diags := LoadConfigForTests(t, rootDir)
	if diags.HasErrors() {
		cleanup()
		t.Fatal(diags.Err())
	}
	return config, loader, cleanup
}
