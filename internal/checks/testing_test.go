// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package checks_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/terraform"
)

// LoadConfigForTests is a test helper that loads a configuration using
// terraform.BuildConfigWithGraph. This helper exists in the checks package
// so that tests can load configs without creating an import cycle.
func LoadConfigForTests(t *testing.T, fixtureDir, testsDir string) *configs.Config {
	t.Helper()

	loader, close := configload.NewLoaderForTests(t)
	defer close()

	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, nil, nil)
	_, instDiags := inst.InstallModules(context.Background(), fixtureDir, testsDir, true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	rootMod, hclDiags := loader.LoadRootModuleWithTests(fixtureDir, testsDir)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid root module: %s", hclDiags.Error())
	}

	cfg, buildDiags := terraform.BuildConfigWithGraph(
		rootMod,
		loader.ModuleWalker(),
		nil, // no input variables
		configs.MockDataLoaderFunc(loader.LoadExternalMockData),
	)
	if buildDiags.HasErrors() {
		t.Fatalf("invalid configuration: %s", buildDiags.Err())
	}

	return cfg
}
