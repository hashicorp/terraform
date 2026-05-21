// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/terraform"
)

// loadRefactoringFixture reads a configuration from the given directory and
// does some naive static processing on any count and for_each expressions
// inside, in order to get a realistic-looking instances.Set for what it
// declares without having to run a full Terraform plan.
func loadRefactoringFixture(t *testing.T, dir string) (*configs.Config, instances.Set) {
	t.Helper()

	loader, cleanup := configload.NewLoaderForTests(t)
	defer cleanup()

	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil), nil)
	_, instDiags := inst.InstallModules(context.Background(), dir, "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	rootMod, diags := loader.LoadRootModule(dir)
	if diags.HasErrors() {
		t.Fatalf("invalid root module: %s", diags.Error())
	}

	rootCfg, buildDiags := terraform.BuildConfigWithGraph(
		rootMod,
		loader.ModuleWalker(),
		nil,
		configs.MockDataLoaderFunc(loader.LoadExternalMockData),
	)
	if buildDiags.HasErrors() {
		t.Fatalf("invalid configuration: %s", buildDiags.Err())
	}

	expander := instances.NewExpander(nil)
	refactoring.StaticPopulateExpanderModule(t, rootCfg, addrs.RootModuleInstance, expander)
	return rootCfg, expander.AllInstances()
}
