// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"context"
	"fmt"
	"log"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/apparentlymart/go-versions/versions"
	
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/registry/regsrc"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ModuleVersionCandidate represents a module version with its provider requirements
// This is used by the constraint solver to represent resolved module versions
type ModuleVersionCandidate struct {
	Module               addrs.ModuleRegistryPackage
	Version              *version.Version
	VersionString        string
	ProviderRequirements providerreqs.Requirements
}

// LazyModuleResolver implements PubGrub-inspired lazy module resolution
// using proven Go patterns from the dep tool, specifically for Terraform modules
type LazyModuleResolver struct {
	installer *ModuleInstaller
	reg       *registry.Client

	// Caches to minimize network calls to module registry
	versionCache    map[string][]*version.Version           // packageAddr -> sorted versions
	dependencyCache map[string]providerreqs.Requirements   // packageAddr@version -> requirements
}

// ModuleSelection represents a module with its selected version and provider dependencies
type ModuleSelection struct {
	ModuleKey           string
	PackageAddr         addrs.ModuleRegistryPackage
	Request             configs.ModuleRequest
	SelectedVersion     *version.Version
	ProviderRequirements providerreqs.Requirements
}

// ModuleResolutionResult contains the final module resolution
type ModuleResolutionResult struct {
	Selections     map[string]*ModuleSelection  // moduleKey -> selection
	Conflicts      []string
	ResolutionPath []string
}

// NewLazyModuleResolver creates a new lazy module resolver for Terraform modules
func NewLazyModuleResolver(installer *ModuleInstaller) *LazyModuleResolver {
	return &LazyModuleResolver{
		installer:       installer,
		reg:             installer.reg,
		versionCache:    make(map[string][]*version.Version),
		dependencyCache: make(map[string]providerreqs.Requirements),
	}
}

// ResolveModules performs PubGrub-inspired module resolution with lazy loading
func (r *LazyModuleResolver) ResolveModules(
	ctx context.Context,
	moduleRequests []configs.ModuleRequest,
) (*ModuleResolutionResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Filter to registry modules only
	registryRequests := r.filterRegistryModules(moduleRequests)
	if len(registryRequests) == 0 {
		return &ModuleResolutionResult{
			Selections:     make(map[string]*ModuleSelection),
			Conflicts:      []string{},
			ResolutionPath: []string{"no registry modules found"},
		}, diags
	}

	log.Printf("[TRACE] LazyModuleResolver: resolving %d registry modules", len(registryRequests))

	// Convert to internal representation
	packages := make(map[string]*packageCandidate)
	for _, req := range registryRequests {
		moduleKey := req.Path.String() + ":" + req.Name
		registryAddr := req.SourceAddr.(addrs.ModuleSourceRegistry)
		
		packages[moduleKey] = &packageCandidate{
			ModuleKey:   moduleKey,
			PackageAddr: registryAddr.Package,
			Request:     req,
			State:       stateUnresolved,
		}
	}

	// Perform resolution using PubGrub-inspired algorithm
	result, resolutionDiags := r.resolveWithPrioritization(ctx, packages)
	diags = diags.Append(resolutionDiags)

	return result, diags
}

// packageState tracks the resolution state of each package
type packageState int

const (
	stateUnresolved packageState = iota
	stateResolving
	stateResolved
	stateConflicted
)

// packageCandidate represents a package being resolved
type packageCandidate struct {
	ModuleKey   string
	PackageAddr addrs.ModuleRegistryPackage
	Request     configs.ModuleRequest
	State       packageState
	
	// Set when resolved
	SelectedVersion      *version.Version
	ProviderRequirements providerreqs.Requirements
}

// resolveWithPrioritization implements the core PubGrub-inspired resolution
func (r *LazyModuleResolver) resolveWithPrioritization(
	ctx context.Context,
	packages map[string]*packageCandidate,
) (*ModuleResolutionResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var resolutionPath []string

	result := &ModuleResolutionResult{
		Selections:     make(map[string]*ModuleSelection),
		Conflicts:      []string{},
		ResolutionPath: resolutionPath,
	}

	// Resolution loop - prioritize packages and resolve incrementally
	for {
		// Find next package to resolve using PubGrub prioritization
		nextPackage := r.selectNextPackage(ctx, packages)
		if nextPackage == nil {
			break // All packages resolved
		}

		log.Printf("[TRACE] LazyModuleResolver: resolving %s", nextPackage.ModuleKey)
		nextPackage.State = stateResolving

		// Choose best version for this package (lazy - only fetch when needed)
		selectedVersion, versionDiags := r.chooseVersion(ctx, nextPackage)
		diags = diags.Append(versionDiags)
		if versionDiags.HasErrors() {
			nextPackage.State = stateConflicted
			result.Conflicts = append(result.Conflicts, 
				fmt.Sprintf("Failed to resolve %s: %s", nextPackage.ModuleKey, versionDiags.Err().Error()))
			continue
		}

		// Get dependencies for the selected version (lazy - only this version)
		dependencies, depDiags := r.getDependencies(ctx, nextPackage.PackageAddr, selectedVersion)
		diags = diags.Append(depDiags)
		if depDiags.HasErrors() {
			nextPackage.State = stateConflicted
			result.Conflicts = append(result.Conflicts, 
				fmt.Sprintf("Failed to get dependencies for %s@%s", nextPackage.ModuleKey, selectedVersion.String()))
			continue
		}

		// Check for provider constraint conflicts using Go dep-style intersection
		conflicts := r.checkProviderConflicts(result.Selections, dependencies)
		if len(conflicts) > 0 {
			nextPackage.State = stateConflicted
			result.Conflicts = append(result.Conflicts, conflicts...)
			log.Printf("[WARN] LazyModuleResolver: conflicts detected for %s@%s", 
				nextPackage.ModuleKey, selectedVersion.String())
			continue
		}

		// Success - mark as resolved
		nextPackage.State = stateResolved
		nextPackage.SelectedVersion = selectedVersion
		nextPackage.ProviderRequirements = dependencies

		selection := &ModuleSelection{
			ModuleKey:            nextPackage.ModuleKey,
			PackageAddr:          nextPackage.PackageAddr,
			Request:              nextPackage.Request,
			SelectedVersion:      selectedVersion,
			ProviderRequirements: dependencies,
		}
		result.Selections[nextPackage.ModuleKey] = selection

		resolutionPath = append(resolutionPath, 
			fmt.Sprintf("resolved %s to %s", nextPackage.ModuleKey, selectedVersion.String()))
		
		log.Printf("[TRACE] LazyModuleResolver: successfully resolved %s to %s", 
			nextPackage.ModuleKey, selectedVersion.String())
	}

	result.ResolutionPath = resolutionPath

	// Determine overall success
	if len(result.Conflicts) == 0 {
		result.ResolutionPath = append(result.ResolutionPath, "resolution completed successfully")
	} else {
		result.ResolutionPath = append(result.ResolutionPath, 
			fmt.Sprintf("resolution completed with %d conflicts", len(result.Conflicts)))
	}

	return result, diags
}

// selectNextPackage implements PubGrub prioritization - resolve constrained packages first
func (r *LazyModuleResolver) selectNextPackage(
	ctx context.Context,
	packages map[string]*packageCandidate,
) *packageCandidate {
	var unresolved []*packageCandidate
	
	for _, pkg := range packages {
		if pkg.State == stateUnresolved {
			unresolved = append(unresolved, pkg)
		}
	}
	
	if len(unresolved) == 0 {
		return nil // All resolved
	}
	
	if len(unresolved) == 1 {
		return unresolved[0]
	}

	// PubGrub prioritization: choose package with fewest satisfying versions
	// This finds conflicts early and reduces search space
	type packagePriority struct {
		pkg           *packageCandidate
		candidateCount int
	}
	
	var priorities []packagePriority
	
	for _, pkg := range unresolved {
		versions, err := r.getAvailableVersions(ctx, pkg.PackageAddr)
		if err != nil {
			// If we can't get versions, give it low priority
			priorities = append(priorities, packagePriority{pkg: pkg, candidateCount: 1000})
			continue
		}
		
		// Count versions that satisfy constraints
		satisfyingCount := 0
		for _, v := range versions {
			if pkg.Request.VersionConstraint.Required.Check(v) {
				satisfyingCount++
			}
		}
		
		priorities = append(priorities, packagePriority{
			pkg:           pkg,
			candidateCount: satisfyingCount,
		})
	}
	
	// Sort by candidate count (fewer = higher priority)
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].candidateCount < priorities[j].candidateCount
	})
	
	selected := priorities[0].pkg
	log.Printf("[TRACE] LazyModuleResolver: prioritized %s (%d candidates)", 
		selected.ModuleKey, priorities[0].candidateCount)
	
	return selected
}

// chooseVersion selects the best version using Go dep-style newest-compatible strategy
func (r *LazyModuleResolver) chooseVersion(
	ctx context.Context,
	pkg *packageCandidate,
) (*version.Version, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	
	versions, err := r.getAvailableVersions(ctx, pkg.PackageAddr)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to get versions",
			fmt.Sprintf("Cannot get versions for %s: %s", pkg.PackageAddr, err),
		))
		return nil, diags
	}
	
	// Find newest version that satisfies constraints (versions already sorted newest first)
	for _, v := range versions {
		if pkg.Request.VersionConstraint.Required.Check(v) {
			log.Printf("[TRACE] LazyModuleResolver: chose %s for %s", v.String(), pkg.ModuleKey)
			return v, diags
		}
	}
	
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"No compatible version",
		fmt.Sprintf("No version of %s satisfies constraint %s", 
			pkg.PackageAddr, pkg.Request.VersionConstraint.Required.String()),
	))
	
	return nil, diags
}

// Helper methods for caching and network operations
func (r *LazyModuleResolver) filterRegistryModules(requests []configs.ModuleRequest) []configs.ModuleRequest {
	var registryRequests []configs.ModuleRequest
	for _, req := range requests {
		if req.SourceAddr != nil {
			if _, isRegistry := req.SourceAddr.(addrs.ModuleSourceRegistry); isRegistry {
				registryRequests = append(registryRequests, req)
			}
		}
	}
	return registryRequests
}

func (r *LazyModuleResolver) getAvailableVersions(
	ctx context.Context,
	packageAddr addrs.ModuleRegistryPackage,
) ([]*version.Version, error) {
	cacheKey := packageAddr.String()
	
	// Check cache
	if versions, exists := r.versionCache[cacheKey]; exists {
		return versions, nil
	}
	
	// Fetch from registry
	regsrcAddr := regsrc.ModuleFromRegistryPackageAddr(packageAddr)
	resp, err := r.reg.ModuleVersions(ctx, regsrcAddr)
	if err != nil {
		return nil, err
	}
	
	if len(resp.Modules) == 0 {
		return nil, fmt.Errorf("no versions found")
	}
	
	var versions []*version.Version
	for _, mv := range resp.Modules[0].Versions {
		if v, err := version.NewVersion(mv.Version); err == nil {
			versions = append(versions, v)
		}
	}
	
	// Sort newest first
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})
	
	// Cache result
	r.versionCache[cacheKey] = versions
	log.Printf("[TRACE] LazyModuleResolver: cached %d versions for %s", len(versions), cacheKey)
	
	return versions, nil
}

func (r *LazyModuleResolver) getDependencies(
	ctx context.Context,
	packageAddr addrs.ModuleRegistryPackage,
	version *version.Version,
) (providerreqs.Requirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	cacheKey := fmt.Sprintf("%s@%s", packageAddr.String(), version.String())
	
	// Check cache
	if deps, exists := r.dependencyCache[cacheKey]; exists {
		return deps, diags
	}
	
	// Use existing analyzer to get dependencies for this specific version
	analyzer := NewModuleProviderAnalyzer(r.installer)
	requirements, analysisDiags := analyzer.extractProviderRequirementsFromModule(
		ctx, packageAddr, version.String())
	diags = diags.Append(analysisDiags)
	
	// Cache result
	r.dependencyCache[cacheKey] = requirements
	log.Printf("[TRACE] LazyModuleResolver: cached dependencies for %s (%d providers)", 
		cacheKey, len(requirements))
	
	return requirements, diags
}

func (r *LazyModuleResolver) checkProviderConflicts(
	existingSelections map[string]*ModuleSelection,
	newRequirements providerreqs.Requirements,
) []string {
	var conflicts []string
	
	// Group all provider requirements by provider
	allRequirements := make(map[addrs.Provider][]providerreqs.VersionConstraints)
	
	// Add existing requirements
	for _, selection := range existingSelections {
		for provider, constraints := range selection.ProviderRequirements {
			allRequirements[provider] = append(allRequirements[provider], constraints)
		}
	}
	
	// Add new requirements
	for provider, constraints := range newRequirements {
		allRequirements[provider] = append(allRequirements[provider], constraints)
	}
	
	// Check for conflicts using Go dep-style constraint intersection
	for provider, constraintsList := range allRequirements {
		if len(constraintsList) <= 1 {
			continue
		}
		
		// Use the existing constraint compatibility check
		if !r.areConstraintsCompatible(constraintsList) {
			conflicts = append(conflicts, 
				fmt.Sprintf("provider %s has incompatible version constraints", provider))
		}
	}
	
	return conflicts
}

// areConstraintsCompatible checks if multiple constraints can be satisfied simultaneously
// Uses the existing logic from the original resolver
func (r *LazyModuleResolver) areConstraintsCompatible(constraints []providerreqs.VersionConstraints) bool {
	if len(constraints) <= 1 {
		return true
	}
	
	// Convert all constraints to version sets and intersect them
	sets := make([]versions.Set, len(constraints))
	for i, constraint := range constraints {
		sets[i] = versions.MeetingConstraints(constraint)
	}
	
	// Find the intersection of all sets
	intersection := versions.Intersection(sets...)
	
	// If the intersection is infinite, it means there are compatible versions
	if !intersection.IsFinite() {
		return true
	}
	
	// If finite, check if there are any versions
	return intersection.List().Len() > 0
}