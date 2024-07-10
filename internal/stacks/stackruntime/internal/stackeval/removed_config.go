// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var _ StaticEvaler = (*RemovedConfig)(nil)

type RemovedConfig struct {
	addr   stackaddrs.ConfigComponent // Since we remove components, we can use the component address
	config *stackconfig.Removed

	main *Main

	validate   promising.Once[tfdiags.Diagnostics]
	moduleTree promising.Once[withDiagnostics[*configs.Config]]
}

func newRemovedConfig(main *Main, config *stackconfig.Removed) *RemovedConfig {
	return &RemovedConfig{
		config: config,
		main:   main,
	}
}

func (c *RemovedConfig) Addr() stackaddrs.ConfigComponent {
	return c.addr
}

// PlanChanges implements Plannable.
func (c *RemovedConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

func (c *RemovedConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	diags, err := c.validate.Do(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		var diags tfdiags.Diagnostics

		moduleTree, moreDiags := c.CheckModuleTree(ctx)
		diags = diags.Append(moreDiags)
		if moduleTree == nil {
			return diags, nil
		}
		decl := c.Declaration(ctx)

		_, providerDiags := c.CheckProviders(ctx, phase)
		diags = diags.Append(providerDiags)
		if providerDiags.HasErrors() {
			// If there's invalid provider configuration, we can't actually go
			// on and validate the module tree. We need the providers and if
			// they're invalid we'll just get crazy and confusing errors
			// later if we try and carry on.
			return diags, nil
		}

		providerSchemas, moreDiags, skipFurtherValidation := c.neededProviderSchemas(ctx, phase)
		if skipFurtherValidation {
			return diags.Append(moreDiags), nil
		}
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return diags, nil
		}

		tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
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

		providerClients, valid := c.neededProviderClients(ctx, phase)
		if !valid {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot validate component",
				Detail:   fmt.Sprintf("Cannot validate %q because its provider configuration assignments are invalid.", c.Addr()),
				Subject:  decl.DeclRange.ToHCL().Ptr(),
			})
			return diags, nil
		}
		defer func() {
			// Close the unconfigured provider clients that we opened in
			// neededProviderClients.
			for _, client := range providerClients {
				client.Close()
			}
		}()

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

// CheckModuleTree loads the tree of Terraform modules starting at the
// component block's configured source address, returning the resulting
// configuration object if successful.
//
// If the module has any problems that prevent even static decoding then
// this instead returns diagnostics and a nil configuration object.
func (c *RemovedConfig) CheckModuleTree(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
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
			diags = diags.Append(c.validateModuleTreeForStacks(configRoot))

			return configRoot, diags
		},
	)
}

func (c *RemovedConfig) Declaration(ctx context.Context) *stackconfig.Removed {
	return c.config
}

func (c *RemovedConfig) CheckProviders(ctx context.Context, phase EvalPhase) (addrs.Set[addrs.RootProviderConfig], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	stackConfig := c.StackConfig(ctx)
	declConfigs := c.Declaration(ctx).ProviderConfigs
	neededProviders := c.RequiredProviderInstances(ctx)

	ret := addrs.MakeSet[addrs.RootProviderConfig]()
	for _, elem := range neededProviders.Elems {

		// sourceAddr is the addrs.RootProviderConfig that should be used to
		// set this provider in the component later.
		sourceAddr := elem.Key

		// componentAddr is the addrs.LocalProviderConfig that specifies the
		// local name and (optional) alias of the provider in the component.
		componentAddr := elem.Value.Local

		// typeAddr is the absolute address of the provider type itself.
		typeAddr := sourceAddr.Provider

		expr, exists := declConfigs[componentAddr]
		if !exists {
			// Then this provider isn't listed in the `providers` block of this
			// component. Which is bad!
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required provider configuration",
				Detail: fmt.Sprintf(
					"The root module for %s requires a provider configuration named %q for provider %q, which is not assigned in the component's \"providers\" argument.",
					c.Addr(), componentAddr.StringCompact(), typeAddr.ForDisplay(),
				),
				Subject: c.Declaration(ctx).DeclRange.ToHCL().Ptr(),
			})
			continue
		}

		// At the validation stage, it's really likely the result here is
		// unknown. But, we can still check the returned type to make sure it
		// matches everything expected.
		result, hclDiags := EvalExprAndEvalContext(ctx, expr, phase, c)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			continue
		}

		// Next, we want to make sure the linked providers are actually of the
		// same type.

		const errSummary = "Invalid provider configuration"
		if actualTy := result.Value.Type(); stackconfigtypes.IsProviderConfigType(actualTy) {
			// Then we at least got a provider reference of some kind.
			actualTypeAddr := stackconfigtypes.ProviderForProviderConfigType(actualTy)
			if actualTypeAddr != typeAddr {
				var errorDetail string

				stackName, matchingTypeExists := stackConfig.ProviderLocalName(ctx, typeAddr)
				_, matchingNameExists := stackConfig.ProviderForLocalName(ctx, componentAddr.LocalName)
				moduleProviderTypeExplicit := elem.Value.Explicit
				if !matchingTypeExists && !matchingNameExists {
					// Then the user just hasn't declared the target provider
					// type or name at all. We'll return a generic error message
					// asking the user to update the required_providers list.
					errorDetail = "\n\nDeclare the required provider in the stack's required_providers block, and then assign a configuration for that provider in this component's \"providers\" argument."
				} else if !matchingNameExists {
					// Then we have a type that matches, but the name doesn't.
					errorDetail = fmt.Sprintf("\n\nThis stack has a configured provider of the correct type under the name %q. Update this component's \"providers\" argument to reference this provider.", stackName)
				} else if !matchingTypeExists {
					// Then we have a name that matches, but the type doesn't.

					// If the types don't match and the names do, then maybe
					// the user hasn't properly filled in the required types
					// within the module.
					if !moduleProviderTypeExplicit {
						// Yes! The provider type within the module has been
						// implied by Terraform and not explicitly set within
						// the required_providers block. We'll suggest the user
						// to update the required_providers block of the module.
						errorDetail = fmt.Sprintf("\n\nThe module does not declare a source address for %q in its required_providers block, so Terraform assumed %q for backward-compatibility with older versions of Terraform", componentAddr.LocalName, elem.Key.Provider.ForDisplay())
					}

					// Otherwise the user has explicitly set the provider type
					// within the module, but it doesn't match the provider type
					// within the stack configuration. The generic error message
					// should be sufficient.
				}

				// But, unfortunately, the underlying types of the providers
				// do not match up.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail: fmt.Sprintf(
						"The provider configuration slot %q requires a configuration for provider %q, not for provider %q.%s",
						componentAddr.StringCompact(), typeAddr, actualTypeAddr, errorDetail,
					),
					Subject: result.Expression.Range().Ptr(),
				})
				continue
			}
		} else if result.Value == cty.DynamicVal {
			// Then we don't know the concrete type of this reference at this
			// time, so we'll just have to accept it. This is somewhat expected
			// during the validation phase, and even during the planning phase
			// if we have deferred attributes. We'll get an error later (ie.
			// during the plan phase) if the type doesn't match up then.
		} else {
			// We got something that isn't a provider reference at all.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"The provider configuration slot %s requires a configuration for provider %q.",
					componentAddr.StringCompact(), typeAddr,
				),
				Subject: result.Expression.Range().Ptr(),
			})
			continue
		}

		// If we made it here, the types all matched up so we've done everything
		// we can. component_instance.go will do additional checks to make sure
		// the result is known and not null when it comes time to actually
		// check the plan.

		ret.Add(sourceAddr)
	}
	return ret, diags
}

func (c *RemovedConfig) StackConfig(ctx context.Context) *StackConfig {
	return c.main.mustStackConfig(ctx, c.addr.Stack)
}

func (c *RemovedConfig) neededProviderClients(ctx context.Context, phase EvalPhase) (map[addrs.RootProviderConfig]providers.Interface, bool) {
	insts := make(map[addrs.RootProviderConfig]providers.Interface)
	valid := true

	providers, _ := c.CheckProviders(ctx, phase)
	for _, provider := range providers {
		pTy := c.main.ProviderType(ctx, provider.Provider)
		if pTy == nil {
			valid = false
			continue // not our job to report a missing provider
		}

		// We don't need to configure the client for validate functionality.
		inst, err := pTy.UnconfiguredClient(ctx)
		if err != nil {
			valid = false
			continue
		}
		insts[provider] = inst
	}

	return insts, valid
}

func (c *RemovedConfig) neededProviderSchemas(ctx context.Context, phase EvalPhase) (map[addrs.Provider]providers.ProviderSchema, tfdiags.Diagnostics, bool) {
	var diags tfdiags.Diagnostics
	skipFutherValidation := false

	config := c.ModuleTree(ctx)
	decl := c.Declaration(ctx)

	providerSchemas := make(map[addrs.Provider]providers.ProviderSchema)
	for _, sourceAddr := range config.ProviderTypes() {
		pTy := c.main.ProviderType(ctx, sourceAddr)
		if pTy == nil {
			continue // not our job to report a missing provider
		}

		// If this phase has a dependency lockfile, check if the provider is in it.
		depLocks := c.main.DependencyLocks(phase)
		if depLocks != nil {
			// Check if the provider is in the lockfile,
			// if it is not we can not read the provider schema
			providerLockfileDiags := CheckProviderInLockfile(*depLocks, pTy, decl.DeclRange)

			// We report these diagnostics in a different place
			if providerLockfileDiags.HasErrors() {
				skipFutherValidation = true
				continue
			}
		}

		schema, err := pTy.Schema(ctx)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider initialization error",
				Detail:   fmt.Sprintf("Failed to fetch the provider schema for %s: %s.", sourceAddr, err),
				Subject:  decl.DeclRange.ToHCL().Ptr(),
			})
			continue
		}
		providerSchemas[sourceAddr] = schema
	}
	return providerSchemas, diags, skipFutherValidation
}

// ModuleTree returns the static representation of the tree of modules starting
// at the component's configured source address, or nil if any of the
// modules have errors that prevent even static decoding.
func (c *RemovedConfig) ModuleTree(ctx context.Context) *configs.Config {
	ret, _ := c.CheckModuleTree(ctx)
	return ret
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
func (c *RemovedConfig) validateModuleTreeForStacks(startNode *configs.Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(c.validateModuleForStacks(startNode.Path, startNode.Module))
	for _, childNode := range startNode.Children {
		diags = diags.Append(c.validateModuleTreeForStacks(childNode))
	}
	return diags
}

func (c *RemovedConfig) validateModuleForStacks(moduleAddr addrs.Module, module *configs.Module) tfdiags.Diagnostics {
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
func (c *RemovedConfig) RequiredProviderInstances(ctx context.Context) addrs.Map[addrs.RootProviderConfig, configs.RequiredProviderConfig] {
	moduleTree := c.ModuleTree(ctx)
	if moduleTree == nil || moduleTree.Root == nil {
		// If we get here then we presumably failed to load the module, and
		// so we'll just unwind quickly so a different return path can return
		// the error diagnostics.
		return addrs.MakeMap[addrs.RootProviderConfig, configs.RequiredProviderConfig]()
	}
	return moduleTree.Root.EffectiveRequiredProviderConfigs()
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *RemovedConfig) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

func (c *RemovedConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if c.Declaration(ctx).ForEach != nil {
		// For validation, we'll return unknown for the instance data.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	return c.StackConfig(ctx).resolveExpressionReference(ctx, ref, nil, repetition)
}

// Validate implements Validatable.
func (c *RemovedConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return c.checkValid(ctx, ValidatePhase)
}

func (c *RemovedConfig) tracingName() string {
	return "RemovedBlock"
}
