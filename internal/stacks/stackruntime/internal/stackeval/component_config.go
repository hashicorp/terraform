// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ComponentConfig struct {
	addr   stackaddrs.ConfigComponent
	config *stackconfig.Component

	main *Main

	moduleTree promising.Once[withDiagnostics[*configs.Config]]
}

func newComponentConfig(main *Main, addr stackaddrs.ConfigComponent, config *stackconfig.Component) *ComponentConfig {
	return &ComponentConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (c *ComponentConfig) Addr() stackaddrs.ConfigComponent {
	return c.addr
}

func (c *ComponentConfig) Declaration(ctx context.Context) *stackconfig.Component {
	return c.config
}

// ModuleTree returns the static representation of the tree of modules starting
// at the component's configured source address, or nil if any of the
// modules have errors that prevent even static decoding.
func (c *ComponentConfig) ModuleTree(ctx context.Context) *configs.Config {
	ret, _ := c.CheckModuleTree(ctx)
	return ret
}

// CheckModuleTree loads the tree of Terraform modules starting at the
// component block's configured source address, returning the resulting
// configuration object if successful.
//
// If the module has any problems that prevent even static decoding then
// this instead returns diagnostics and a nil configuration object.
func (c *ComponentConfig) CheckModuleTree(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, &c.moduleTree, c.main,
		func(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			decl := c.Declaration(ctx)
			sources := c.main.SourceBundle(ctx)

			// The "configs" package predates the idea of explicit source
			// bundles, so for now we need to do some adaptation to
			// help it interact with the files in the source bundle despite
			// not being aware of that abstraction.
			// TODO: Introduce source bundle support into the "configs" package
			// API, and factor out some of this complexity onto there.

			rootModuleSource := decl.FinalSourceAddr
			if rootModuleSource == nil {
				// If we get here then the configuration was loaded incorrectly,
				// either by the stackconfig package or by the caller of the
				// stackconfig package using the wrong loading function.
				panic("component configuration lacks final source address")
			}
			rootModuleDir, err := sources.LocalPathForSource(rootModuleSource)
			if err != nil {
				// We should not get here if the source bundle was constructed
				// correctly.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Can't load module for component",
					Detail:   fmt.Sprintf("Failed to load this component's root module: %s.", tfdiags.FormatError(err)),
					Subject:  decl.SourceAddrRange.ToHCL().Ptr(),
				})
				return nil, diags
			}

			// Since the module config loader doesn't yet understand source
			// bundles, any diagnostics we return from here will contain the
			// real filesystem path of the problematic file rather than
			// preserving the source bundle abstraction. As a compromise
			// though, we'll make the path relative to the current working
			// directory so at least it won't be quite so obnoxiously long
			// when we're running in situations like a remote executor that
			// uses a separate directory per job.
			// FIXME: Make the module loader aware of source bundles and use
			// source addresses in its diagnostics, etc.
			if cwd, err := os.Getwd(); err == nil {
				relPath, err := filepath.Rel(cwd, rootModuleDir)
				if err == nil {
					rootModuleDir = filepath.ToSlash(relPath)
				}
			}

			// With rootModuleDir we can now have the configs package work
			// directly with the real filesystem, rather than with the source
			// bundle. However, this does mean that any error messages generated
			// from this process will disclose the real locations of the
			// source files on disk (an implementation detail) rather than
			// preserving the source address abstraction.
			parser := configs.NewParser(afero.NewOsFs())
			parser.AllowLanguageExperiments(c.main.LanguageExperimentsAllowed())

			if !parser.IsConfigDir(rootModuleDir) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Can't load module for component",
					Detail:   fmt.Sprintf("The source location %s does not contain a Terraform module.", rootModuleSource),
					Subject:  decl.SourceAddrRange.ToHCL().Ptr(),
				})
				return nil, diags
			}

			rootMod, hclDiags := parser.LoadConfigDir(rootModuleDir)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				return nil, diags
			}

			configRoot, hclDiags := configs.BuildConfig(rootMod, &sourceBundleModuleWalker{
				sources: sources,
				parser:  parser,
			}, nil)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				return nil, diags
			}

			// We also have a small selection of additional static validation
			// rules that apply only to modules used within stack components.
			diags = diags.Append(c.validateModuleTreeForStacks(configRoot))

			return configRoot, diags
		},
	)
}

// validateModuleTreeForStacks imposes some additional validation constraints
// on a module tree after it's been loaded by the main configuration packages.
//
// These rules deal with a small number of exceptions where the modules language
// as used by stacks is a subset of the modules language from traditional
// Terraform. Not all such exceptions are handled in this way because
// some of them cannot be handled statically, but this is a reasonable place
// to handle the simpler concerns and allows us to return error messages that
// talk specifically about stacks, which would be harder to achieve if these
// exceptions were made at a different layer.
func (c *ComponentConfig) validateModuleTreeForStacks(startNode *configs.Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(c.validateModuleForStacks(startNode.Path, startNode.Module))
	for _, childNode := range startNode.Children {
		diags = diags.Append(c.validateModuleTreeForStacks(childNode))
	}
	return diags
}

func (c *ComponentConfig) validateModuleForStacks(moduleAddr addrs.Module, module *configs.Module) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Inline provider configurations are not allowed when running under stacks,
	// because provider configurations live in the stack configuration and
	// then get passed in to the modules as special arguments.
	for _, pc := range module.ProviderConfigs {
		// We use some slightly different language for the topmost module
		// that's being directly called from the stack configuration, because
		// we can give some direct advice for how to correct the problem there,
		// whereas for a nested module we assume that it's a third-party module
		// written for much older versions of Terraform before we deprecated
		// inline provider configurations and thus the solution is most likely
		// to be selecting a different module that is Stacks-compatible, because
		// removing a legacy inline provider configuration from a shared module
		// would be a breaking change to that module.
		if moduleAddr.IsRoot() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Inline provider configuration not allowed",
				Detail:   "A module used as a stack component must have all of its provider configurations passed from the stack configuration, using the \"providers\" argument within the component configuration block.",
				Subject:  pc.DeclRange.Ptr(),
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Inline provider configuration not allowed",
				Detail:   "This module is not compatible with Terraform Stacks, because it declares an inline provider configuration.\n\nTo be used with stacks, this module must instead accept provider configurations from its caller.",
				Subject:  pc.DeclRange.Ptr(),
			})
		}
	}

	return diags
}

// InputsType returns an object type that the object representing the caller's
// values for this component's input variables must conform to.
func (c *ComponentConfig) InputsType(ctx context.Context) (cty.Type, *typeexpr.Defaults) {
	moduleTree := c.ModuleTree(ctx)
	if moduleTree == nil {
		// If the module tree is invalid itself then we can't determine which
		// input variables are declared.
		return cty.NilType, nil
	}

	vars := moduleTree.Module.Variables
	atys := make(map[string]cty.Type, len(vars))
	defs := &typeexpr.Defaults{
		DefaultValues: make(map[string]cty.Value),
		Children:      map[string]*typeexpr.Defaults{},
	}
	var opts []string
	for name, v := range vars {
		atys[name] = v.ConstraintType
		if def := v.Default; def != cty.NilVal {
			defs.DefaultValues[name] = def
			opts = append(opts, name)
		}
		if childDefs := v.TypeDefaults; childDefs != nil {
			defs.Children[name] = childDefs
		}
	}
	retTy := cty.ObjectWithOptionalAttrs(atys, opts)
	defs.Type = retTy
	return retTy, defs
}

// RequiredProviderInstances returns a description of all of the provider
// instance slots ("provider configurations" in main Terraform language
// terminology) that are either explicitly declared or implied by the
// root module of the component's module tree.
//
// The component configuration must include a "providers" argument that
// binds each of these slots to a real provider instance in the stack
// configuration, by referring to dynamic values of the appropriate
// provider instance reference type.
//
// In the returned map the keys describe provider configurations from
// the perspective of an object inside the root module, and so the LocalName
// field values are an implementation detail that must not be exposed into
// the calling stack and are included here only so that we can potentially
// return error messages referring to declarations inside the module.
//
// If any modules in the component's root module tree are invalid then this
// result could under-promise or over-promise depending on the kind of
// invalidity.
func (c *ComponentConfig) RequiredProviderInstances(ctx context.Context) addrs.Set[addrs.RootProviderConfig] {
	moduleTree := c.ModuleTree(ctx)
	if moduleTree == nil || moduleTree.Root == nil {
		// If we get here then we presumably failed to load the module, and
		// so we'll just unwind quickly so a different return path can return
		// the error diagnostics.
		return nil
	}
	return moduleTree.Root.EffectiveRequiredProviderConfigs()
}

// ExprReferenceValue implements Referenceable.
func (c *ComponentConfig) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	// Currently we don't say anything at all about component results during
	// validation, since the main Terraform language's validate call doesn't
	// return any information about hypothetical root module output values.
	// We don't expose ComponentConfig in any scope outside of the validation
	// phase, so this is sufficient for all phases. (See [Component] for how
	// component results get calculated during the plan and apply phases.)
	return cty.DynamicVal
}

func (c *ComponentConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckModuleTree(ctx)
	diags = diags.Append(moreDiags)

	return diags
}

// Validate implements Validatable.
func (c *ComponentConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return c.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (c *ComponentConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

func (c *ComponentConfig) tracingName() string {
	return c.Addr().String()
}

// sourceBundleModuleWalker is an implementation of [configs.ModuleWalker]
// that loads all modules from a single source bundle.
type sourceBundleModuleWalker struct {
	sources *sourcebundle.Bundle
	parser  *configs.Parser
}

// LoadModule implements configs.ModuleWalker.
func (w *sourceBundleModuleWalker) LoadModule(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	// First we need to assemble the "final source address" for the module
	// by asking the source bundle to match the given source address and
	// version against what's in the bundle manifest. This should cause
	// use to make the same decision that the source bundler made about
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

	moduleDir, err := w.sources.LocalPathForSource(finalSourceAddr)
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

	// If the moduleDir is relative then it's relative to the parent module's
	// source directory, we'll make it absolute.
	if !filepath.IsAbs(moduleDir) {
		moduleDir = filepath.Clean(filepath.Join(req.Parent.Module.SourceDir, moduleDir))
	}

	// Since the module config loader doesn't yet understand source
	// bundles, any diagnostics we return from here will contain the
	// real filesystem path of the problematic file rather than
	// preserving the source bundle abstraction. As a compromise
	// though, we'll make the path relative to the current working
	// directory so at least it won't be quite so obnoxiously long
	// when we're running in situations like a remote executor that
	// uses a separate directory per job.
	// FIXME: Make the module loader aware of source bundles and use
	// source addresses in its diagnostics, etc.
	if cwd, err := os.Getwd(); err == nil {
		relPath, err := filepath.Rel(cwd, moduleDir)
		if err == nil {
			moduleDir = filepath.ToSlash(relPath)
		}
	}

	mod, moreDiags := w.parser.LoadConfigDir(moduleDir)
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

func (w *sourceBundleModuleWalker) finalSourceForModule(tfSourceAddr addrs.ModuleSource, versionConstraints *version.Constraints) (sourceaddrs.FinalSource, error) {
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

func (w *sourceBundleModuleWalker) bundleSourceAddrForTerraformSourceAddr(tfSourceAddr addrs.ModuleSource) (sourceaddrs.Source, error) {
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

func (w *sourceBundleModuleWalker) versionSetForLegacyVersionConstraints(versionConstraints *version.Constraints) (versions.Set, error) {
	// In practice this should always succeed because the source bundle builder
	// would've parsed the same version constraints using this same parser
	// and so source bundle building would've failed if the given address were
	// outside the subset supported for source bundles.
	return versions.MeetingConstraintsStringRuby(versionConstraints.String())
}

func (w *sourceBundleModuleWalker) legacyVersionForVersion(v versions.Version) (*version.Version, error) {
	return version.NewVersion(v.String())
}
