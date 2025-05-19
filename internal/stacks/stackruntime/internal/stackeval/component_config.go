// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

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
	stackparser "github.com/hashicorp/terraform/internal/stacks/stackconfig/parser"
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
	stack  *StackConfig
	config *stackconfig.Component

	main *Main

	validate   perEvalPhase[promising.Once[tfdiags.Diagnostics]]
	moduleTree promising.Once[withDiagnostics[*configs.Config]] // moduleTree is constant across all phases
}

func newComponentConfig(main *Main, addr stackaddrs.ConfigComponent, stack *StackConfig, config *stackconfig.Component) *ComponentConfig {
	return &ComponentConfig{
		addr:   addr,
		stack:  stack,
		config: config,
		main:   main,
	}
}

// Addr implements ConfigComponentExpressionScope
func (c *ComponentConfig) Addr() stackaddrs.ConfigComponent {
	return c.addr
}

// DeclRange implements ConfigComponentExpressionScope
func (c *ComponentConfig) DeclRange() *hcl.Range {
	return c.config.DeclRange.ToHCL().Ptr()
}

// StackConfig implements ConfigComponentExpressionScope
func (c *ComponentConfig) StackConfig() *StackConfig {
	return c.stack
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
		ctx, c.tracingName()+" modules", &c.moduleTree,
		func(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			sources := c.main.SourceBundle()

			rootModuleSource := c.config.FinalSourceAddr
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
					Subject:  c.config.SourceAddrRange.ToHCL().Ptr(),
				})
				return nil, diags
			}

			rootMod, hclDiags := parser.LoadConfigDir(rootModuleSource)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				return nil, diags
			}

			walker := stackparser.NewSourceBundleModuleWalker(rootModuleSource, sources, parser)
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

	varDecls := c.RootModuleVariableDecls(ctx)

	// We don't care about the returned value, only that it has no errors.
	_, diags := EvalComponentInputVariables(ctx, varDecls, wantTy, defs, c.config, phase, c)
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

	// By calling `checkValid` on ourself here, we will cause a cycle error to be exposed if we ended
	// up within this function while executing c.checkValid initially. This just makes sure that there
	// are no cycles between components.
	c.checkValid(ctx, phase)
	return cty.DynamicVal
}

// ResolveExpressionReference implements ExpressionScope.
func (c *ComponentConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if c.config.ForEach != nil {
		// For validation, we'll return unknown for the instance data.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	return c.stack.resolveExpressionReference(ctx, ref, nil, repetition)
}

// ExternalFunctions implements ExpressionScope.
func (c *ComponentConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.stack)
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *ComponentConfig) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

func (c *ComponentConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	diags, err := c.validate.For(phase).Do(ctx, c.tracingName(), func(ctx context.Context) (tfdiags.Diagnostics, error) {
		var diags tfdiags.Diagnostics

		moduleTree, moreDiags := c.CheckModuleTree(ctx)
		diags = diags.Append(moreDiags)
		if moduleTree == nil {
			return diags, nil
		}

		variableDiags := c.CheckInputVariableValues(ctx, phase)
		diags = diags.Append(variableDiags)

		dependsOnDiags := ValidateDependsOn(c.stack, c.config.DependsOn)
		diags = diags.Append(dependsOnDiags)

		// We don't actually exit if we found errors with the input variables
		// or depends_on attribute, we can still validate the actual module tree
		// without them.

		providerTypes, providerDiags := EvalProviderTypes(ctx, c.stack, c.config.ProviderConfigs, phase, c)
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
				provider, err := c.main.ProviderType(addr).UnconfiguredClient()
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

		providerClients, valid := unconfiguredProviderClients(c.main, providerTypes)
		if !valid {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot validate component",
				Detail:   fmt.Sprintf("Cannot validate %s because its provider configuration assignments are invalid.", c.addr),
				Subject:  c.config.DeclRange.ToHCL().Ptr(),
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
	switch err := err.(type) {
	case promising.ErrSelfDependent:
		// This is a case where the component is self-dependent, which is
		// a cycle that we can't resolve. We'll report this as a diagnostic
		// and then continue on to report any other diagnostics that we found.
		// The promise reporter is main, so that we can get the names of all promises
		// involved in the cycle.
		diags = diags.Append(diagnosticsForPromisingTaskError(err))
	default:
		if err != nil {
			// this is crazy, we never return an error from the inner function so
			// this really shouldn't happen.
			panic(fmt.Sprintf("unexpected error from validate.Do: %s", err))
		}
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
	return c.addr.String()
}
