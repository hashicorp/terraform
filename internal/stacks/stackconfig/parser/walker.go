// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package parser

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// SourceBundleModuleWalker is an implementation of [configs.ModuleWalker]
// that loads all modules from a single source bundle.
type SourceBundleModuleWalker struct {
	absoluteSourceAddrs map[string]sourceaddrs.FinalSource
	sources             *sourcebundle.Bundle
	parser              *configs.SourceBundleParser
}

func NewSourceBundleModuleWalker(rootModuleSource sourceaddrs.FinalSource, sources *sourcebundle.Bundle, parser *configs.SourceBundleParser) *SourceBundleModuleWalker {
	absoluteSourceAddrs := make(map[string]sourceaddrs.FinalSource, 1)
	absoluteSourceAddrs[addrs.RootModule.String()] = rootModuleSource
	return &SourceBundleModuleWalker{
		absoluteSourceAddrs: absoluteSourceAddrs,
		sources:             sources,
		parser:              parser,
	}
}

// LoadModule implements configs.ModuleWalker.
func (w *SourceBundleModuleWalker) LoadModule(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	// First we need to assemble the "final source address" for the module
	// by asking the source bundle to match the given source address and
	// version against what's in the bundle manifest. This should cause
	// us to make the same decision that the source bundler made about
	// which real package to use.
	finalSourceAddr, err := w.finalSourceForModule(req.SourceAddr, &req.VersionConstraint.Required)
	if err != nil {
		// We should not typically get here because we're translating
		// Terraform's own source address representations to the same
		// representations the source bundle builder would've used, but
		// we'll be robust about it nonetheless.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Can't load module for component",
			Detail:   fmt.Sprintf("Invalid source address: %s.", err),
			Subject:  req.SourceAddrRange.Ptr(),
		})
		return nil, nil, diags
	}

	absoluteSourceAddr, err := w.absoluteSourceAddr(finalSourceAddr, req.Parent)
	if err != nil {
		// Again, this should not happen, but let's ensure we can debug if it
		// does.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Can't load module for component",
			Detail:   fmt.Sprintf("Unable to determine absolute source address: %s.", err),
			Subject:  req.SourceAddrRange.Ptr(),
		})
		return nil, nil, diags
	}

	// We store the absolute source address for this module so that any in-repo
	// child modules can use it to construct their absolute source addresses
	// too.
	w.absoluteSourceAddrs[req.Path.String()] = absoluteSourceAddr

	_, err = w.sources.LocalPathForSource(absoluteSourceAddr)
	if err != nil {
		// We should not get here if the source bundle was constructed
		// correctly.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Can't load module for component",
			Detail:   fmt.Sprintf("Failed to load this component's module %s: %s.", req.Path.String(), tfdiags.FormatError(err)),
			Subject:  req.SourceAddrRange.Ptr(),
		})
		return nil, nil, diags
	}

	mod, moreDiags := w.parser.LoadConfigDir(absoluteSourceAddr)
	diags = append(diags, moreDiags...)

	// Annoyingly we now need to translate our version selection back into
	// the legacy type again, so we can return it through the ModuleWalker API.
	var legacyV *version.Version
	if modSrc, ok := finalSourceAddr.(sourceaddrs.RegistrySourceFinal); ok {
		legacyV, err = w.legacyVersionForVersion(modSrc.SelectedVersion())
		if err != nil {
			// It would be very strange to get in here because by now we've
			// already round-tripped between the legacy and modern version
			// constraint representations once, so we should have a version
			// number that's compatible with both.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Can't load module for component",
				Detail:   fmt.Sprintf("Invalid version string %q: %s.", modSrc.SelectedVersion(), err),
				Subject:  req.SourceAddrRange.Ptr(),
			})
		}
	}
	return mod, legacyV, diags
}

func (w *SourceBundleModuleWalker) finalSourceForModule(tfSourceAddr addrs.ModuleSource, versionConstraints *version.Constraints) (sourceaddrs.FinalSource, error) {
	// Unfortunately the configs package still uses our old model of version
	// constraints and Terraform's own form of source addresses, so we need
	// to adapt to what the sourcebundle API is expecting.
	sourceAddr, err := w.bundleSourceAddrForTerraformSourceAddr(tfSourceAddr)
	if err != nil {
		return nil, err
	}
	var allowedVersions versions.Set
	if versionConstraints != nil {
		allowedVersions, err = w.versionSetForLegacyVersionConstraints(versionConstraints)
		if err != nil {
			return nil, fmt.Errorf("invalid version constraints: %w", err)
		}
	} else {
		allowedVersions = versions.Released
	}

	switch sourceAddr := sourceAddr.(type) {
	case sourceaddrs.FinalSource:
		// Most source address types are already final source addresses.
		return sourceAddr, nil
	case sourceaddrs.RegistrySource:
		// Registry sources are trickier because we need to figure out which
		// exact version we're using.
		vs := w.sources.RegistryPackageVersions(sourceAddr.Package())
		v := vs.NewestInSet(allowedVersions)
		return sourceAddr.Versioned(v), nil
	default:
		// Should not get here because the above should be exhaustive for all
		// possible address types.
		return nil, fmt.Errorf("unsupported source address type %T", tfSourceAddr)
	}
}

func (w *SourceBundleModuleWalker) bundleSourceAddrForTerraformSourceAddr(tfSourceAddr addrs.ModuleSource) (sourceaddrs.Source, error) {
	// In practice this should always succeed because the source bundle builder
	// would've parsed the same source addresses using these same parsers
	// and so source bundle building would've failed if the given address were
	// outside the subset supported for source bundles.
	switch tfSourceAddr := tfSourceAddr.(type) {
	case addrs.ModuleSourceLocal:
		return sourceaddrs.ParseLocalSource(tfSourceAddr.String())
	case addrs.ModuleSourceRemote:
		return sourceaddrs.ParseRemoteSource(tfSourceAddr.String())
	case addrs.ModuleSourceRegistry:
		return sourceaddrs.ParseRegistrySource(tfSourceAddr.String())
	default:
		// Should not get here because the above should be exhaustive for all
		// possible address types.
		return nil, fmt.Errorf("unsupported source address type %T", tfSourceAddr)
	}
}

func (w *SourceBundleModuleWalker) absoluteSourceAddr(sourceAddr sourceaddrs.FinalSource, parent *configs.Config) (sourceaddrs.FinalSource, error) {
	switch source := sourceAddr.(type) {
	case sourceaddrs.LocalSource:
		parentPath := addrs.RootModule
		if parent != nil {
			parentPath = parent.Path
		}
		absoluteParentSourceAddr, ok := w.absoluteSourceAddrs[parentPath.String()]
		if !ok {
			return nil, fmt.Errorf("unexpected missing source address for module parent %q", parentPath)
		}
		return sourceaddrs.ResolveRelativeFinalSource(absoluteParentSourceAddr, source)
	default:
		return sourceAddr, nil
	}
}

func (w *SourceBundleModuleWalker) versionSetForLegacyVersionConstraints(versionConstraints *version.Constraints) (versions.Set, error) {
	// In practice this should always succeed because the source bundle builder
	// would've parsed the same version constraints using this same parser
	// and so source bundle building would've failed if the given address were
	// outside the subset supported for source bundles.
	return versions.MeetingConstraintsStringRuby(versionConstraints.String())
}

func (w *SourceBundleModuleWalker) legacyVersionForVersion(v versions.Version) (*version.Version, error) {
	return version.NewVersion(v.String())
}
