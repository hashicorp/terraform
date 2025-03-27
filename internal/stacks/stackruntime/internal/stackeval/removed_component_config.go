// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	stackparser "github.com/hashicorp/terraform/internal/stacks/stackconfig/parser"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Validatable                                                = (*RemovedComponentConfig)(nil)
	_ Plannable                                                  = (*RemovedComponentConfig)(nil)
	_ ExpressionScope                                            = (*RemovedComponentConfig)(nil)
	_ ConfigComponentExpressionScope[stackaddrs.ConfigComponent] = (*RemovedComponentConfig)(nil)
)

type RemovedComponentConfig struct {
	addr   stackaddrs.ConfigComponent
	config *stackconfig.Removed
	stack  *StackConfig

	main *Main

	validate   perEvalPhase[promising.Once[tfdiags.Diagnostics]]
	moduleTree promising.Once[withDiagnostics[*configs.Config]] // moduleTree is constant for every phase
}

func newRemovedComponentConfig(main *Main, addr stackaddrs.ConfigComponent, stack *StackConfig, config *stackconfig.Removed) *RemovedComponentConfig {
	return &RemovedComponentConfig{
		addr:   addr,
		config: config,
		stack:  stack,
		main:   main,
	}
}

// Addr implements ConfigComponentExpressionScope.
func (r *RemovedComponentConfig) Addr() stackaddrs.ConfigComponent {
	return r.addr
}

// DeclRange implements ConfigComponentExpressionScope.
func (r *RemovedComponentConfig) DeclRange() *hcl.Range {
	return r.config.DeclRange.ToHCL().Ptr()
}

// StackConfig implements ConfigComponentExpressionScope
func (r *RemovedComponentConfig) StackConfig() *StackConfig {
	return r.stack
}

// ModuleTree implements ConfigComponentExpressionScope
func (r *RemovedComponentConfig) ModuleTree(ctx context.Context) *configs.Config {
	cfg, _ := r.CheckModuleTree(ctx)
	return cfg
}

// CheckModuleTree loads and validates the module tree for the component that
// is being removed.
func (r *RemovedComponentConfig) CheckModuleTree(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" modules", &r.moduleTree, func(ctx context.Context) (*configs.Config, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics

		decl := r.config
		sources := r.main.SourceBundle()

		rootModuleSource := decl.FinalSourceAddr
		if rootModuleSource == nil {
			// If we get here then the configuration was loaded incorrectly,
			// either by the stackconfig package or by the caller of the
			// stackconfig package using the wrong loading function.
			panic("component configuration lacks final source address")
		}

		parser := configs.NewSourceBundleParser(sources)
		parser.AllowLanguageExperiments(r.main.LanguageExperimentsAllowed())

		if !parser.IsConfigDir(rootModuleSource) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Can't load module for removed component",
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
	})
}

// CheckValid validates the module tree and provider configurations for the
// component being removed.
func (r *RemovedComponentConfig) CheckValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	diags, err := r.validate.For(phase).Do(ctx, r.tracingName(), func(ctx context.Context) (tfdiags.Diagnostics, error) {
		var diags tfdiags.Diagnostics

		moduleTree, moreDiags := r.CheckModuleTree(ctx)
		diags = diags.Append(moreDiags)
		if moduleTree == nil {
			return diags, nil
		}

		providers, moreDiags := EvalProviderTypes(ctx, r.stack, r.config.ProviderConfigs, phase, r)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return diags, nil
		}

		providerSchemas, moreDiags, skipFurtherValidation := neededProviderSchemas(ctx, r.main, phase, r)
		if skipFurtherValidation {
			return diags.Append(moreDiags), nil
		}
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return diags, nil
		}

		tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
			PreloadedProviderSchemas: providerSchemas,
			Provisioners:             r.main.availableProvisioners(),
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

		providerClients, valid := unconfiguredProviderClients(r.main, providers)
		if !valid {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot validate component",
				Detail:   fmt.Sprintf("Cannot validate %s because its provider configuration assignments are invalid.", r.Addr()),
				Subject:  r.DeclRange(),
			})
			return diags, nil
		}
		defer func() {
			// Close the unconfigured provider clients that we opened in
			// unconfiguredProviderClients.
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

// PlanChanges implements Plannable.
func (r *RemovedComponentConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, r.CheckValid(ctx, PlanPhase)
}

// Validate implements Validatable.
func (r *RemovedComponentConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return r.CheckValid(ctx, ValidatePhase)
}

// tracingName implements tracingNamer.
func (r *RemovedComponentConfig) tracingName() string {
	return fmt.Sprintf("%s (removed)", r.Addr())
}

// ResolveExpressionReference implements ExpressionScope.
func (r *RemovedComponentConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if r.config.ForEach != nil {
		// For validation, we'll return unknown for the instance data.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	return r.stack.resolveExpressionReference(ctx, ref, nil, repetition)
}

// PlanTimestamp implements ExpressionScope.
func (r *RemovedComponentConfig) PlanTimestamp() time.Time {
	return r.main.PlanTimestamp()
}

// ExternalFunctions implements ExpressionScope.
func (r *RemovedComponentConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.main.ProviderFunctions(ctx, r.stack)
}
