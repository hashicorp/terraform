// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/getmodules"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/registry/regsrc"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ModuleProviderAnalyzer handles analysis of individual module versions for provider requirements
// This is used by the lazy dependency provider to analyze specific versions on-demand
type ModuleProviderAnalyzer struct {
	installer *ModuleInstaller
	loader    *configload.Loader
	reg       *registry.Client
	fetcher   *getmodules.PackageFetcher
}

// NewModuleProviderAnalyzer creates a new analyzer instance
func NewModuleProviderAnalyzer(installer *ModuleInstaller) *ModuleProviderAnalyzer {
	return &ModuleProviderAnalyzer{
		installer: installer,
		loader:    installer.loader,
		reg:       installer.reg,
		fetcher:   getmodules.NewPackageFetcher(),
	}
}

// extractProviderRequirementsFromModule downloads and analyzes a specific module version
// This is the core functionality used by the lazy dependency provider
func (r *ModuleProviderAnalyzer) extractProviderRequirementsFromModule(
	ctx context.Context,
	packageAddr addrs.ModuleRegistryPackage,
	versionStr string,
) (providerreqs.Requirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	requirements := make(providerreqs.Requirements)
	
	// Create a temporary directory for downloading this version
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("terraform-module-analysis-%s-%s", packageAddr.Name, versionStr))
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create temporary directory",
			fmt.Sprintf("Could not create temporary directory for module analysis: %s", err),
		))
		return requirements, diags
	}
	defer os.RemoveAll(tempDir)
	
	// Get download URL from registry
	regsrcAddr := regsrc.ModuleFromRegistryPackageAddr(packageAddr)
	realAddrRaw, err := r.reg.ModuleLocation(ctx, regsrcAddr, versionStr)
	if err != nil {
		log.Printf("[WARN] ModuleProviderAnalyzer: failed to get download URL for %s@%s: %s", packageAddr, versionStr, err)
		return requirements, diags
	}
	
	// Download the module using the raw URL directly
	err = r.fetcher.FetchPackage(ctx, tempDir, realAddrRaw)
	if err != nil {
		log.Printf("[WARN] ModuleProviderAnalyzer: failed to download %s@%s: %s", packageAddr, versionStr, err)
		return requirements, diags
	}
	
	// Parse the downloaded module to extract provider requirements
	moduleRequirements, parseDiags := r.parseModuleForProviders(tempDir)
	diags = diags.Append(parseDiags)
	
	// Merge the requirements
	for provider, constraint := range moduleRequirements {
		requirements[provider] = constraint
	}
	
	log.Printf("[TRACE] ModuleProviderAnalyzer: extracted %d provider requirements from %s@%s", 
		len(requirements), packageAddr, versionStr)
	
	return requirements, diags
}

// parseModuleForProviders parses a downloaded module and extracts provider requirements
func (r *ModuleProviderAnalyzer) parseModuleForProviders(moduleDir string) (providerreqs.Requirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	requirements := make(providerreqs.Requirements)
	
	// Look for Terraform configuration files
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read module directory",
			fmt.Sprintf("Could not read module directory %s: %s", moduleDir, err),
		))
		return requirements, diags
	}
	
	parser := hclparse.NewParser()
	
	// Parse all .tf and .tf.json files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".tf") && !strings.HasSuffix(filename, ".tf.json") {
			continue
		}
		
		fullPath := filepath.Join(moduleDir, filename)
		
		var file *hcl.File
		var hclDiags hcl.Diagnostics
		
		if strings.HasSuffix(filename, ".tf.json") {
			file, hclDiags = parser.ParseJSONFile(fullPath)
		} else {
			file, hclDiags = parser.ParseHCLFile(fullPath)
		}
		
		if hclDiags.HasErrors() {
			// Log but don't fail - some files might have issues but we can still extract providers
			log.Printf("[WARN] ModuleProviderAnalyzer: failed to parse %s: %s", fullPath, hclDiags.Error())
			continue
		}
		
		// Extract provider requirements from this file
		fileRequirements := r.extractProvidersFromHCL(file)
		
		// Merge requirements
		for provider, constraint := range fileRequirements {
			if existing, exists := requirements[provider]; exists {
				// If we already have a constraint for this provider, we need to intersect them
				// For now, just use the more restrictive one (this is a simplification)
				log.Printf("[TRACE] ModuleProviderAnalyzer: found multiple constraints for %s, using existing", provider)
				requirements[provider] = existing
			} else {
				requirements[provider] = constraint
			}
		}
	}
	
	return requirements, diags
}

// extractProvidersFromHCL extracts provider requirements from an HCL file
func (r *ModuleProviderAnalyzer) extractProvidersFromHCL(file *hcl.File) providerreqs.Requirements {
	requirements := make(providerreqs.Requirements)
	
	if file.Body == nil {
		return requirements
	}
	
	// Look for terraform blocks with required_providers
	content, _, _ := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type: "terraform",
			},
		},
	})
	
	for _, block := range content.Blocks {
		if block.Type != "terraform" {
			continue
		}
		
		terraformContent, _, _ := block.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type: "required_providers",
				},
			},
		})
		
		for _, reqProvidersBlock := range terraformContent.Blocks {
			attrs, _ := reqProvidersBlock.Body.JustAttributes()
			
			for name, attr := range attrs {
				// Parse provider requirement
				providerAddr, constraint, err := r.parseProviderRequirement(name, attr)
				if err == nil && providerAddr != nil && constraint != nil {
					requirements[*providerAddr] = constraint
				}
			}
		}
	}
	
	return requirements
}

// parseProviderRequirement parses a single provider requirement attribute
func (r *ModuleProviderAnalyzer) parseProviderRequirement(name string, attr *hcl.Attribute) (*addrs.Provider, providerreqs.VersionConstraints, error) {
	// This is a simplified parser - in practice, provider requirements can be complex
	// For now, we'll extract basic version constraints
	
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return nil, nil, fmt.Errorf("failed to evaluate provider requirement")
	}
	
	// Handle simple string version constraint like: aws = "~> 5.0"
	if val.Type() == cty.String {
		versionStr := val.AsString()
		
		// Parse the provider address (assume registry.terraform.io/hashicorp/name format)
		providerAddr := addrs.NewDefaultProvider(name)
		
		// Parse version constraint
		constraint, err := providerreqs.ParseVersionConstraints(versionStr)
		if err != nil {
			log.Printf("[WARN] ModuleProviderAnalyzer: failed to parse version constraint %s for %s: %s", versionStr, name, err)
			return nil, nil, err
		}
		
		return &providerAddr, constraint, nil
	}
	
	// Handle object format like: aws = { source = "hashicorp/aws", version = "~> 5.0" }
	if val.Type().IsObjectType() {
		obj := val.AsValueMap()
		
		var providerAddr addrs.Provider
		var versionStr string
		
		// Extract source
		if sourceVal, hasSource := obj["source"]; hasSource && sourceVal.Type() == cty.String {
			source := sourceVal.AsString()
			parsedAddr, err := addrs.ParseProviderSourceString(source)
			if err == nil {
				providerAddr = parsedAddr
			}
		}
		
		// Extract version
		if versionVal, hasVersion := obj["version"]; hasVersion && versionVal.Type() == cty.String {
			versionStr = versionVal.AsString()
		}
		
		// If we don't have a source, use default
		if providerAddr.IsZero() {
			providerAddr = addrs.NewDefaultProvider(name)
		}
		
		// Parse version constraint
		if versionStr != "" {
			constraint, err := providerreqs.ParseVersionConstraints(versionStr)
			if err != nil {
				log.Printf("[WARN] ModuleProviderAnalyzer: failed to parse version constraint %s for %s: %s", versionStr, name, err)
				return nil, nil, err
			}
			
			return &providerAddr, constraint, nil
		}
	}
	
	return nil, nil, fmt.Errorf("unsupported provider requirement format")
}