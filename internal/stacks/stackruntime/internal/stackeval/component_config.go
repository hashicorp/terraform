// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Validatable                                                = (*ComponentConfig)(nil)
	_ Plannable                                                  = (*ComponentConfig)(nil)
	_ ExpressionScope                                            = (*ComponentConfig)(nil)
	_ ConfigComponentExpressionScope[stackaddrs.ConfigComponent] = (*ComponentConfig)(nil)
)

type ComponentConfig struct {
	addr   stackaddrs.ConfigComponent
	config *stackconfig.Component

	main *Main

	validate   promising.Once[tfdiags.Diagnostics]
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

func (c *ComponentConfig) DeclRange(_ context.Context) *hcl.Range {
	return c.config.DeclRange.ToHCL().Ptr()
}

func (c *ComponentConfig) StackConfig(ctx context.Context) *StackConfig {
	return c.main.mustStackConfig(ctx, c.addr.Stack)
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

			rootModuleSource := decl.FinalSourceAddr
			if rootModuleSource == nil {
				// If we get here then the configuration was loaded incorrectly,
				// either by the stackconfig package or by the caller of the
				// stackconfig package using the wrong loading function.
				panic("component configuration lacks final source address")
			}

			parser := configs.NewSourceBundleParser(sources)
			parser.AllowLanguageExperiments(c.main.LanguageExperimentsAllowed())

			if !parser.IsConfigDir(rootModuleSource) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Can't load module for component",
					Detail:   fmt.Sprintf("The source location %s does not contain a Terraform module.", rootModuleSource),
					Subject:  decl.SourceAddrRange.ToHCL().Ptr(),
				})
				return nil, diags
			}

			rootMod, hclDiags := parser.LoadConfigDir(rootModuleSource)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				return nil, diags
			}

			walker := newSourceBundleModuleWalker(rootModuleSource, sources, parser)
			configRoot, hclDiags := configs.BuildConfig(rootMod, walker, nil)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				return nil, diags
			}

			// We also have a small selection of additional static validation
			// rules that apply only to modules used within stack components.
			diags = diags.Append(validateModuleTreeForStacks(configRoot))

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
func validateModuleTreeForStacks(startNode *configs.Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(validateModuleForStacks(startNode.Path, startNode.Module))
	for _, childNode := range startNode.Children {
		diags = diags.Append(validateModuleTreeForStacks(childNode))
	}
	return diags
}

func validateModuleForStacks(moduleAddr addrs.Module, module *configs.Module) tfdiags.Diagnostics {
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

func (c *ComponentConfig) RootModuleVariableDecls(ctx context.Context) map[string]*configs.Variable {
	moduleTree := c.ModuleTree(ctx)
	if moduleTree == nil {
		// If the module tree is invalid then we'll just assume there aren't
		// any variables declared.
		return nil
	}
	return moduleTree.Module.Variables
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

func (c *ComponentConfig) CheckInputVariableValues(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	wantTy, defs := c.InputsType(ctx)
	if wantTy == cty.NilType {
		// Suggests that the module tree is invalid. We validate the full module
		// tree elsewhere, which will hopefully detect the problems here.
		return nil
	}

	decl := c.Declaration(ctx)
	varDecls := c.RootModuleVariableDecls(ctx)

	// We don't care about the returned value, only that it has no errors.
	_, diags := EvalComponentInputVariables(ctx, varDecls, wantTy, defs, decl, phase, c)
	return diags
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

// ResolveExpressionReference implements ExpressionScope.
func (c *ComponentConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if c.Declaration(ctx).ForEach != nil {
		// For validation, we'll return unknown for the instance data.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	return c.StackConfig(ctx).resolveExpressionReference(ctx, ref, nil, repetition)
}

// ExternalFunctions implements ExpressionScope.
func (c *ComponentConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.StackConfig(ctx))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *ComponentConfig) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

func (c *ComponentConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	diags, err := c.validate.Do(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		var diags tfdiags.Diagnostics

		moduleTree, moreDiags := c.CheckModuleTree(ctx)
		diags = diags.Append(moreDiags)
		if moduleTree == nil {
			return diags, nil
		}
		decl := c.Declaration(ctx)

		variableDiags := c.CheckInputVariableValues(ctx, phase)
		diags = diags.Append(variableDiags)

		dependsOnDiags := ValidateDependsOn(ctx, c.StackConfig(ctx), c.config.DependsOn)
		diags = diags.Append(dependsOnDiags)

		// We don't actually exit if we found errors with the input variables
		// or depends_on attribute, we can still validate the actual module tree
		// without them.

		providerTypes, providerDiags := EvalProviderTypes(ctx, c.StackConfig(ctx), c.config.ProviderConfigs, phase, c)
		diags = diags.Append(providerDiags)
		if providerDiags.HasErrors() {
			// If there's invalid provider configuration, we can't actually go
			// on and validate the module tree. We need the providers and if
			// they're invalid we'll just get crazy and confusing errors
			// later if we try and carry on.
			return diags, nil
		}

		providerSchemas, moreDiags, skipFurtherValidation := neededProviderSchemas(ctx, c.main, phase, c)
		if skipFurtherValidation {
			return diags.Append(moreDiags), nil
		}
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return diags, nil
		}

		providerFactories := make(map[addrs.Provider]providers.Factory, len(providerSchemas))
		for addr := range providerSchemas {
			providerFactories[addr] = func() (providers.Interface, error) {
				// Lazily fetch the unconfigured client for the provider
				// as and when we need it.
				provider, err := c.main.ProviderType(ctx, addr).UnconfiguredClient()
				if err != nil {
					return nil, err
				}
				// this provider should only be used for selected operations
				return stubs.OfflineProvider(provider), nil
			}
		}

		tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
			Providers:                providerFactories,
			PreloadedProviderSchemas: providerSchemas,
			Provisioners:             c.main.availableProvisioners(),
		})
		if err != nil {
			// Should not get here because we should always pass a valid
			// ContextOpts above.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to instantiate Terraform modules runtime",
				fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
			))
			return diags, nil
		}

		providerClients, valid := unconfiguredProviderClients(ctx, c.main, providerTypes)
		if !valid {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot validate component",
				Detail:   fmt.Sprintf("Cannot validate %s because its provider configuration assignments are invalid.", c.Addr()),
				Subject:  decl.DeclRange.ToHCL().Ptr(),
			})
			return diags, nil
		}

		// When our given context is cancelled, we want to instruct the
		// modules runtime to stop the running operation. We use this
		// nested context to ensure that we don't leak a goroutine when the
		// parent context isn't cancelled.
		operationCtx, operationCancel := context.WithCancel(ctx)
		defer operationCancel()
		go func() {
			<-operationCtx.Done()
			if ctx.Err() == context.Canceled {
				tfCtx.Stop()
			}
		}()

		diags = diags.Append(tfCtx.Validate(moduleTree, &terraform.ValidateOpts{
			ExternalProviders: providerClients,
		}))
		return diags, nil
	})
	if err != nil {
		// this is crazy, we never return an error from the inner function so
		// this really shouldn't happen.
		panic(fmt.Sprintf("unexpected error from validate.Do: %s", err))
	}

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

// reportNamedPromises implements namedPromiseReporter.
func (c *ComponentConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(c.validate.PromiseID(), c.Addr().String())
	cb(c.moduleTree.PromiseID(), c.Addr().String()+" modules")
}

// sourceBundleModuleWalker is an implementation of [configs.ModuleWalker]
// that loads all modules from a single source bundle.
type sourceBundleModuleWalker struct {
	absoluteSourceAddrs map[string]sourceaddrs.FinalSource
	sources             *sourcebundle.Bundle
	parser              *configs.SourceBundleParser
}

func newSourceBundleModuleWalker(rootModuleSource sourceaddrs.FinalSource, sources *sourcebundle.Bundle, parser *configs.SourceBundleParser) *sourceBundleModuleWalker {
	absoluteSourceAddrs := make(map[string]sourceaddrs.FinalSource, 1)
	absoluteSourceAddrs[addrs.RootModule.String()] = rootModuleSource
	return &sourceBundleModuleWalker{
		absoluteSourceAddrs: absoluteSourceAddrs,
		sources:             sources,
		parser:              parser,
	}
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

	absoluteSourceAddr, err := w.absoluteSourceAddr(finalSourceAddr, req.Parent)
	if err != nil {
		// Again, this should not happen, but let's ensure we can debug if it
		// does.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Can't load module for component",
			Detail:   fmt.Sprintf("Unable to determin absolute source address: %s.", err),
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

func (w *sourceBundleModuleWalker) absoluteSourceAddr(sourceAddr sourceaddrs.FinalSource, parent *configs.Config) (sourceaddrs.FinalSource, error) {
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
