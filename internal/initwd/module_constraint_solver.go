// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"context"
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ModuleConstraintSolver handles smart resolution of registry module version conflicts
// using PubGrub-inspired lazy constraint satisfaction techniques
type ModuleConstraintSolver struct {
	installer *ModuleInstaller
}

// ConflictResolution contains the results of conflict resolution
type ConflictResolution struct {
	ResolvedVersions map[string]*ModuleVersionCandidate // moduleKey -> resolved candidate
	Conflicts        []string                            // list of unresolvable conflicts
	ResolutionPath   []string                            // description of resolution steps
}

// NewModuleConstraintSolver creates a new registry module resolver
func NewModuleConstraintSolver(installer *ModuleInstaller) *ModuleConstraintSolver {
	return &ModuleConstraintSolver{
		installer: installer,
	}
}

// ResolveRegistryModules performs PubGrub-inspired lazy constraint resolution
func (r *ModuleConstraintSolver) ResolveRegistryModules(
	ctx context.Context,
	moduleRequests []configs.ModuleRequest,
) (*ConflictResolution, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	
	log.Printf("[TRACE] ModuleConstraintSolver: using PubGrub-inspired lazy resolution for %d modules", len(moduleRequests))
	
	// Use the new lazy module resolver for efficient resolution
	lazyResolver := NewLazyModuleResolver(r.installer)
	lazyResult, lazyDiags := lazyResolver.ResolveModules(ctx, moduleRequests)
	diags = diags.Append(lazyDiags)
	
	// Convert lazy result to the expected ConflictResolution format
	resolution := &ConflictResolution{
		ResolvedVersions: make(map[string]*ModuleVersionCandidate),
		Conflicts:        lazyResult.Conflicts,
		ResolutionPath:   lazyResult.ResolutionPath,
	}
	
	// Convert selections to ModuleVersionCandidate format
	for moduleKey, selection := range lazyResult.Selections {
		candidate := &ModuleVersionCandidate{
			Module:               selection.PackageAddr,
			Version:              selection.SelectedVersion,
			VersionString:        selection.SelectedVersion.String(),
			ProviderRequirements: selection.ProviderRequirements,
		}
		resolution.ResolvedVersions[moduleKey] = candidate
	}
	
	log.Printf("[TRACE] ModuleConstraintSolver: lazy resolution completed with %d selections, %d conflicts", 
		len(resolution.ResolvedVersions), len(resolution.Conflicts))
	
	return resolution, diags
}

