// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LoadConfigForTests is a convenience wrapper around configload.NewLoaderForTests,
// initwd.ModuleInstaller.InstallModules and terraform.BuildConfigWithGraph that
// allows a test configuration to be loaded in a single step using the graph-based
// configuration loading mechanism.
//
// If module installation fails, t.Fatal (or similar) is called to halt
// execution of the test, under the assumption that installation failures are
// not expected. If installation failures _are_ expected then use
// configload.NewLoaderForTests and work with the loader object directly. If
// module installation succeeds but generates warnings, these warnings are
// discarded.
//
// If installation succeeds but errors are detected during loading then a
// possibly-incomplete config is returned along with error diagnostics. The
// test run is not aborted in this case, so that the caller can make assertions
// against the returned diagnostics.
//
// As with configload.NewLoaderForTests, a cleanup function is returned which
// must be called before the test completes in order to remove the temporary
// modules directory.
func LoadConfigForTests(t *testing.T, rootDir string, testsDir string) (*configs.Config, *configload.Loader, func(), tfdiags.Diagnostics) {
	t.Helper()

	var diags tfdiags.Diagnostics

	loader, cleanup := configload.NewLoaderForTests(t)
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil), nil)

	_, moreDiags := inst.InstallModules(context.Background(), rootDir, testsDir, true, false, initwd.ModuleInstallHooksImpl{})
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		cleanup()
		t.Fatal(diags.Err())
		return nil, nil, func() {}, diags
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	rootMod, hclDiags := loader.LoadRootModuleWithTests(rootDir, testsDir)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, loader, cleanup, diags
	}

	config, buildDiags := terraform.BuildConfigWithGraph(
		rootMod,
		loader.ModuleWalker(),
		nil, // No input variables for test configs
		configs.MockDataLoaderFunc(loader.LoadExternalMockData),
	)
	diags = diags.Append(buildDiags)

	return config, loader, cleanup, diags
}

// MustLoadConfigForTests is a variant of LoadConfigForTests which calls
// t.Fatal (or similar) if there are any errors during loading, and thus
// does not return diagnostics at all.
//
// This is useful for concisely writing tests that don't expect errors at
// all. For tests that expect errors and need to assert against them, use
// LoadConfigForTests instead.
func MustLoadConfigForTests(t *testing.T, rootDir, testsDir string) (*configs.Config, *configload.Loader, func()) {
	t.Helper()

	config, loader, cleanup, diags := LoadConfigForTests(t, rootDir, testsDir)
	if diags.HasErrors() {
		cleanup()
		t.Fatal(diags.Err())
	}
	return config, loader, cleanup
}
