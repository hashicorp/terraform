// Copyright IBM Corp. 2014, 2026
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
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type EphemeralComponentInstance struct {
	call     *EphemeralComponent
	addr     stackaddrs.AbsEphemeralComponentInstance
	mode     plans.Mode
	deferred bool

	main *Main

	repetition instances.RepetitionData

	resultValue         perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	inputVariableValues perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Applyable = (*EphemeralComponentInstance)(nil)
var _ Plannable = (*EphemeralComponentInstance)(nil)
var _ ExpressionScope = (*EphemeralComponentInstance)(nil)
var _ ConfigComponentExpressionScope[stackaddrs.AbsEphemeralComponentInstance] = (*EphemeralComponentInstance)(nil)

func newEphemeralComponentInstance(call *EphemeralComponent, addr stackaddrs.AbsEphemeralComponentInstance, repetition instances.RepetitionData, mode plans.Mode, deferred bool) *EphemeralComponentInstance {
	return &EphemeralComponentInstance{
		call:       call,
		addr:       addr,
		mode:       mode,
		deferred:   deferred,
		main:       call.main,
		repetition: repetition,
	}
}

func (c *EphemeralComponentInstance) RepetitionData() instances.RepetitionData {
	return c.repetition
}

func (c *EphemeralComponentInstance) InputVariableValues(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckInputVariableValues(ctx, phase)
	return ret
}

func (c *EphemeralComponentInstance) CheckInputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, c.tracingName()+" inputs", c.inputVariableValues.For(phase), func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		config := c.call.config
		wantTy, defs := config.InputsType(ctx)
		varDecls := config.RootModuleVariableDecls(ctx)
		decl := c.call.config.config

		if wantTy == cty.NilType {
			// Suggests that the target module is invalid in some way, so we'll
			// just report that we don't know the input variable values and trust
			// that the module's problems will be reported by some other return
			// path.
			return cty.DynamicVal, nil
		}

		// We actually checked the errors statically already, so we only care about
		// the value here.
		val, diags := EvalComponentInputVariables(ctx, varDecls, wantTy, defs, decl.Inputs, decl.DeclRange, phase, c)
		if diags.HasErrors() {
			return cty.NilVal, diags
		}
		return val, diags
	})
}

// inputValuesForModulesRuntime adapts the result of [EphemeralComponentInstance.InputVariableValues]
// to the representation that the main Terraform modules runtime expects.
func (c *EphemeralComponentInstance) inputValuesForModulesRuntime(ctx context.Context, phase EvalPhase) terraform.InputValues {
	valsObj := c.InputVariableValues(ctx, phase)
	if valsObj == cty.NilVal {
		return nil
	}

	// valsObj might be an unknown value during the planning phase, in which
	// case we'll return an InputValues with all of the expected variables
	// defined as unknown values of their expected type constraints. To
	// achieve that, we'll do our work with the configuration's object type
	// constraint instead of with the value we've been given directly.
	wantTy, _ := c.call.config.InputsType(ctx)
	if wantTy == cty.NilType {
		// The configuration is too invalid for us to know what type we're
		// expecting, so we'll just bail.
		return nil
	}
	wantAttrs := wantTy.AttributeTypes()
	ret := make(terraform.InputValues, len(wantAttrs))
	for name, aty := range wantAttrs {
		v := valsObj.GetAttr(name)
		if !v.IsKnown() {
			// We'll ensure that it has the expected type even if
			// InputVariableValues didn't know what types to use.
			v = cty.UnknownVal(aty)
		}
		ret[name] = &terraform.InputValue{
			Value:      v,
			SourceType: terraform.ValueFromCaller,
		}
	}
	return ret
}

func (c *EphemeralComponentInstance) PlanOpts(ctx context.Context, phase EvalPhase) (*terraform.PlanOpts, tfdiags.Diagnostics) {
	decl := c.call.config.config

	inputValues := c.inputValuesForModulesRuntime(ctx, phase)
	if inputValues == nil {
		return nil, nil
	}

	known, unknown, moreDiags := EvalProviderValues(ctx, c.main, decl.ProviderConfigs, phase, c)
	if moreDiags.HasErrors() {
		// We won't actually add the diagnostics here, they should be
		// exposed via a different return path.
		var diags tfdiags.Diagnostics
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot evaluate ephemeral_component",
			Detail:   fmt.Sprintf("Cannot retrieve ephemeral data for %s because its provider configuration assignments are invalid.", c.Addr()),
			Subject:  decl.DeclRange.ToHCL().Ptr(),
		})
	}

	providerClients := configuredProviderClients(ctx, c.main, known, unknown, phase)

	plantimestamp := c.main.PlanTimestamp()

	return &terraform.PlanOpts{
		// As we're just retrieving ephemeral outputs, we always use normal plan mode.
		Mode:                       plans.NormalMode,
		SetVariables:               inputValues,
		ExternalProviders:          providerClients,
		ExternalDependencyDeferred: c.deferred,
		DeferralAllowed:            true,
		// Ephemeral root outputs are valid for ephemeral components as we have already
		// verified that no managed resources exist in the module tree configuration,
		// which enables us to order the ephemeral components so they do not create cycles
		// during destroy.
		AllowRootEphemeralOutputs: true,
		PolicyClient:              c.main.PolicyClient(),

		// We want the same plantimestamp between all components and the stacks language
		ForcePlanTimestamp: &plantimestamp,
	}, nil
}

func (c *EphemeralComponentInstance) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	val, _ := c.CheckResultValue(ctx, phase)
	return val
}

func (c *EphemeralComponentInstance) CheckResultValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := c.checkResultValue(ctx, phase)

	// Each output carries its own ephemeral mark, but we mark the whole result
	// so that the placeholder unknown values returned when the component is
	// deferred or invalid can never be treated as non-ephemeral.
	return val.Mark(marks.Ephemeral), diags
}

func (c *EphemeralComponentInstance) checkResultValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, c.tracingName(), c.resultValue.For(phase),
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			if phase == ApplyPhase && c.call.requiredComponentsPlanIncompleteInApply(ctx) {
				// A required component wasn't fully planned, so it won't be
				// applied and it's output values can't be used as inputs.
				// Consumers of this ephemeral component will be deferred for the
				// same reason, so this unknown result prevents any additional errors.
				return cty.DynamicVal, nil
			}

			switch phase {
			// Ephemeral components produce result values the same way, by using the plan operation to produce output values
			case PlanPhase, ApplyPhase:
				var diags tfdiags.Diagnostics
				// TODO:@austinvalle: Add hooks for terraform module runtime? There are ephemeral resource ones, but not sure if they are useful?
				// h := hooksFromContext(ctx)

				opts, moreDiags := c.PlanOpts(ctx, phase)
				diags = diags.Append(moreDiags)
				if opts == nil {
					return cty.DynamicVal, diags
				}

				// We don't produce ephemeral data until all of the components we depend on have complete plans.
				if phase == PlanPhase && c.call.requiredComponentsPlanIncompleteInPlan(ctx) {
					opts.ExternalDependencyDeferred = true
				}

				// The instance is upstream deferred if the for_each value for this instance or any parent stack is unknown.
				if c.addr.Item.Key == addrs.WildcardKey {
					opts.ExternalDependencyDeferred = true
				} else {
					for _, step := range c.call.addr.Stack {
						if step.Key == addrs.WildcardKey {
							opts.ExternalDependencyDeferred = true
							break
						}
					}
				}

				moduleTree := c.ModuleTree(ctx)
				if moduleTree == nil {
					// Presumably the configuration is invalid in some way, so we can't create
					// a plan and the relevant diagnostics will get reported when the plan driver
					// visits the [EphemeralComponentConfig] object.
					return cty.DynamicVal, diags
				}

				providerSchemas, moreDiags, _ := neededProviderSchemas(ctx, c.main, phase, c)
				diags = diags.Append(moreDiags)
				if moreDiags.HasErrors() {
					return cty.DynamicVal, diags
				}

				// These factories are used by Terraform core to perform operations that require an
				// unconfigured provider (like executing a provider-defined function). The configured provider
				// clients have already been populated in the PlanOpts argument.
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
					// Should not get here because we should always pass a valid ContextOpts above.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to instantiate Terraform modules runtime",
						fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
					))
					return cty.DynamicVal, diags
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

				// Ephemeral components don't produce a plan or state, however they do need to evaluate the configuration with
				// configured providers, so we always use the plan operation to achieve that.
				_, planScope, moreDiags := tfCtx.PlanAndEval(moduleTree, nil, opts, addrs.RootModuleInstance)
				diags = diags.Append(moreDiags)

				ephOutputVals := ephemeralOutputValuesFromScope(moduleTree, planScope)

				// TODO:@austinvalle: Eventually we need to switch to using the normal "Plan" method and whatever
				// concurrency pipeline we use to return ephemeral data while keeping the actual plan operation running (so TF core will renew/close).
				// c.main.RegisterCleanup(func(ctx context.Context) tfdiags.Diagnostics {
				// 	return nil
				// })

				return cty.ObjectVal(ephOutputVals), diags

			default:
				// Ephemeral components don't produce concrete values outside of the plan or apply phases
				// as they run the plan operation which requires configured providers.
				return cty.DynamicVal, nil
			}
		},
	))
}

func ephemeralOutputValuesFromScope(config *configs.Config, scope *lang.Scope) map[string]cty.Value {
	if config == nil || scope == nil {
		return nil
	}

	ephOutputVals := make(map[string]cty.Value, 0)
	for _, oc := range config.Module.Outputs {
		if !oc.Ephemeral {
			continue
		}

		// The only diagnostic that can be returned here is if the output address
		// doesn't exist, which should be impossible here.
		v, _ := scope.Data.GetOutput(oc.Addr(), tfdiags.SourceRangeFromHCL(oc.DeclRange))
		ephOutputVals[oc.Name] = v
	}

	return ephOutputVals
}

func (c *EphemeralComponentInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := c.call.stack
	return stack.resolveExpressionReference(ctx, ref, nil, c.repetition)
}

func (c *EphemeralComponentInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.call.config.stack)
}

func (c *EphemeralComponentInstance) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

func (c *EphemeralComponentInstance) Addr() stackaddrs.AbsEphemeralComponentInstance {
	return c.addr
}

func (c *EphemeralComponentInstance) StackConfig() *StackConfig {
	return c.call.stack.config
}

func (c *EphemeralComponentInstance) ModuleTree(ctx context.Context) *configs.Config {
	return c.call.config.ModuleTree(ctx)
}

func (c *EphemeralComponentInstance) DeclRange() *hcl.Range {
	return c.call.config.config.DeclRange.ToHCL().Ptr()
}

func (c *EphemeralComponentInstance) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckInputVariableValues(ctx, phase)
	diags = diags.Append(moreDiags)

	_, moreDiags = c.CheckResultValue(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

func (c *EphemeralComponentInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

func (c *EphemeralComponentInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

func (c *EphemeralComponentInstance) tracingName() string {
	return c.Addr().String()
}
