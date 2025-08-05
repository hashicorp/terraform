// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/registry"
)

func TestModuleConstraintSolver_ResolveRegistryModules_empty(t *testing.T) {
	loader, close := configload.NewLoaderForTests(t)
	defer close()
	
	inst := NewModuleInstaller("", loader, registry.NewClient(nil, nil))
	solver := NewModuleConstraintSolver(inst)

	// Test with empty module requests
	result, diags := solver.ResolveRegistryModules(t.Context(), []configs.ModuleRequest{})
	
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	
	if len(result.ResolvedVersions) != 0 {
		t.Errorf("expected 0 resolved versions, got %d", len(result.ResolvedVersions))
	}
	
	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(result.Conflicts))
	}
	
	if len(result.ResolutionPath) == 0 {
		t.Error("expected resolution path to have entries")
	}
}

func TestNewModuleConstraintSolver(t *testing.T) {
	loader, close := configload.NewLoaderForTests(t)
	defer close()
	
	inst := NewModuleInstaller("", loader, registry.NewClient(nil, nil))
	solver := NewModuleConstraintSolver(inst)
	
	if solver == nil {
		t.Fatal("expected solver, got nil")
	}
	
	if solver.installer != inst {
		t.Error("expected solver to reference the installer")
	}
}