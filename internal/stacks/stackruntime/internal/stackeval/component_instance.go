package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ComponentInstance struct {
	call *Component
	key  addrs.InstanceKey

	main *Main

	repetition instances.RepetitionData

	moduleTreePlan promising.Once[withDiagnostics[*plans.Plan]]
}

var _ Plannable = (*ComponentInstance)(nil)
var _ ExpressionScope = (*ComponentInstance)(nil)

func newComponentInstance(call *Component, key addrs.InstanceKey, repetition instances.RepetitionData) *ComponentInstance {
	return &ComponentInstance{
		call:       call,
		key:        key,
		main:       call.main,
		repetition: repetition,
	}
}

func (c *ComponentInstance) Addr() stackaddrs.AbsComponentInstance {
	callAddr := c.call.Addr()
	stackAddr := callAddr.Stack
	return stackaddrs.AbsComponentInstance{
		Stack: stackAddr,
		Item: stackaddrs.ComponentInstance{
			Component: callAddr.Item,
			Key:       c.key,
		},
	}
}

func (c *ComponentInstance) InputVariableValues(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckInputVariableValues(ctx, phase)
	return ret
}

func (c *ComponentInstance) CheckInputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	wantTy, defs := c.call.Config(ctx).InputsType(ctx)
	decl := c.call.Declaration(ctx)

	if wantTy == cty.NilType {
		// Suggests that the target module is invalid in some way, so we'll
		// just report that we don't know the input variable values and trust
		// that the module's problems will be reported by some other return
		// path.
		return cty.DynamicVal, diags
	}

	v := cty.EmptyObjectVal
	expr := decl.Inputs
	rng := decl.DeclRange
	var hclCtx *hcl.EvalContext
	if expr != nil {
		result, moreDiags := EvalExprAndEvalContext(ctx, expr, phase, c)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
		expr = result.Expression
		hclCtx = result.EvalContext
		v = result.Value
		rng = tfdiags.SourceRangeFromHCL(result.Expression.Range())
	}

	if defs != nil {
		v = defs.Apply(v)
	}
	v, err := convert.Convert(v, wantTy)
	if err != nil {
		// A conversion failure here could either be caused by an author-provided
		// expression that's invalid or by the author omitting the argument
		// altogether when there's at least one required attribute, so we'll
		// return slightly different messages in each case.
		if expr != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid inputs for component",
				Detail:      fmt.Sprintf("Invalid input variable definition object: %s.", tfdiags.FormatError(err)),
				Subject:     rng.ToHCL().Ptr(),
				Expression:  expr,
				EvalContext: hclCtx,
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required inputs for component",
				Detail:   fmt.Sprintf("Must provide \"inputs\" argument to define the component's input variables: %s.", tfdiags.FormatError(err)),
				Subject:  rng.ToHCL().Ptr(),
			})
		}
		return cty.DynamicVal, diags
	}

	for _, path := range stackconfigtypes.ProviderInstancePathsInValue(v) {
		err := path.NewErrorf("cannot send provider configuration reference to Terraform module input variable")
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid inputs for component",
			Detail: fmt.Sprintf(
				"Invalid input variable definition object: %s.\n\nUse the separate \"providers\" argument to specify the provider configurations to use for this component's root module.",
				tfdiags.FormatError(err),
			),
			Subject:     rng.ToHCL().Ptr(),
			Expression:  expr,
			EvalContext: hclCtx,
		})
	}

	return v, diags
}

// inputValuesForModulesRuntime adapts the result of
// [ComponentInstance.InputVariableValues] to the representation that the
// main Terraform modules runtime expects.
func (c *ComponentInstance) inputValuesForModulesRuntime(ctx context.Context, phase EvalPhase) terraform.InputValues {
	valsObj := c.InputVariableValues(ctx, phase)
	if valsObj == cty.NilVal {
		return nil
	}

	// valsObj might be an unknown value during the planning phase, in which
	// case we'll return an InputValues with all of the expected variables
	// defined as unknown values of their expected type constraints. To
	// achieve that, we'll do our work with the configuration's object type
	// constraint instead of with the value we've been given directly.
	wantTy, _ := c.call.Config(ctx).InputsType(ctx)
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

// CheckProviders evaluates the "providers" argument from the component
// configuration and returns a mapping from the provider configuration
// addresses that the component's root module expect to have populated
// to the address of the [ProviderInstance] from the stack configuration
// to pass into that slot.
//
// If the second return value "valid" is true then the providers argument
// is valid and so the returned map should be complete. If "valid" is false
// then there are some problems with the providers argument and so the
// map might be incomplete, and so callers should use it only with a great
// deal of care.
func (c *ComponentInstance) Providers(ctx context.Context, phase EvalPhase) (selections map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, valid bool) {
	ret, diags := c.CheckProviders(ctx, phase)
	return ret, !diags.HasErrors()
}

// CheckProviders evaluates the "providers" argument from the component
// configuration and returns a mapping from the provider configuration
// addresses that the component's root module expect to have populated
// to the address of the [ProviderInstance] from the stack configuration
// to pass into that slot.
//
// If the "providers" argument is invalid then this will return error
// diagnostics along with a partial result.
func (c *ComponentInstance) CheckProviders(ctx context.Context, phase EvalPhase) (map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance)

	stack := c.call.Stack(ctx)
	stackConfig := stack.StackConfig(ctx)
	declConfigs := c.call.Declaration(ctx).ProviderConfigs
	neededConfigs := c.call.Config(ctx).RequiredProviderInstances(ctx)
	for _, inCalleeAddr := range neededConfigs {
		// declConfigs is based on _local_ provider references so we'll
		// need to translate based on the stack configuration's
		// required_providers block.
		typeAddr := inCalleeAddr.Provider
		localName, ok := stackConfig.ProviderLocalName(ctx, typeAddr)
		if !ok {
			// TODO: We should probably catch this as a one-time error during
			// validation of the component config block, rather than raising
			// it separately for each instance, since the set of required
			// providers for both this stack and the root module of the
			// component are statically-declared.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Component requires undeclared provider",
				Detail: fmt.Sprintf(
					"The root module for %s requires a configuration for provider %q, which isn't declared as a dependency of this stack configuration.\n\nDeclare this provider in the stack's required_providers block, and then assign a configuration for that provider in this component's \"providers\" argument.",
					c.Addr(), typeAddr.ForDisplay(),
				),
				Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
			})
			continue
		}
		localAddr := addrs.LocalProviderConfig{
			LocalName: localName,
			Alias:     inCalleeAddr.Alias,
		}
		expr, ok := declConfigs[localAddr]
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required provider configuration",
				Detail: fmt.Sprintf(
					"The root module for %s requires provider configuration named %q for provider %q, which is not assigned in the component's \"providers\" argument.",
					c.Addr(), localAddr.StringCompact(), typeAddr.ForDisplay(),
				),
				Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
			})
			continue
		}

		// If we've got this far then expr is an expression that should
		// evaluate to a special cty capsule type that acts as a reference
		// to a provider configuration declared elsewhere in the tree
		// of stack configurations.
		result, hclDiags := EvalExprAndEvalContext(ctx, expr, phase, c)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			continue
		}

		const errSummary = "Invalid provider reference"
		if actualTy := result.Value.Type(); stackconfigtypes.IsProviderConfigType(actualTy) {
			actualTypeAddr := stackconfigtypes.ProviderForProviderConfigType(actualTy)
			if actualTypeAddr != typeAddr {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail: fmt.Sprintf(
						"The provider configuration slot %s requires a configuration for provider %q, not for provider %q.",
						localAddr.StringCompact(), typeAddr, actualTypeAddr,
					),
					Subject: result.Expression.Range().Ptr(),
				})
				continue
			}
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"The provider configuration slot %s requires a configuration for provider %q.",
					localAddr.StringCompact(), typeAddr,
				),
				Subject: result.Expression.Range().Ptr(),
			})
		}
		v := result.Value

		// If the tests succeeded above then "v" should definitely
		// be of the expected type, but might be unknown or null.
		if v.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"The provider configuration slot %s is required, but this definition returned null.",
					localAddr.StringCompact(),
				),
				Subject: result.Expression.Range().Ptr(),
			})
			continue
		}
		if !v.IsKnown() {
			// TODO: Once we support deferred changes we should return
			// something that lets the caller know the configuration is
			// incomplete so it can defer planning the entire component.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"This expression depends on values that won't be known until the apply phase, so Terraform cannot determine which provider configuration to use while planning changes for %s.",
					c.Addr().String(),
				),
				Subject: result.Expression.Range().Ptr(),
			})
			continue
		}

		// If it's of the correct type, known, and not null then we should
		// be able to retrieve a specific provider instance address that
		// this value refers to.
		providerInstAddr := stackconfigtypes.ProviderInstanceForValue(v)
		ret[inCalleeAddr] = providerInstAddr

		// The reference must be to a provider instance that's actually
		// configured.
		providerInstStack := c.main.Stack(ctx, providerInstAddr.Stack, phase)
		if providerInstStack != nil {
			provider := providerInstStack.Provider(ctx, providerInstAddr.Item.ProviderConfig)
			if provider != nil {
				insts := provider.Instances(ctx, phase)
				if insts == nil {
					// If we get here then we don't yet know which instances
					// this provider has, so we'll be optimistic that it'll
					// show up in a later phase.
					continue
				}
				if _, exists := insts[providerInstAddr.Item.Key]; exists {
					continue
				}
			}
		}
		// If we fall here then something on the path to the provider instance
		// doesn't exist, and so effectively the provider instance doesn't exist.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail: fmt.Sprintf(
				"Expression result refers to undefined provider instance %s.",
				providerInstAddr,
			),
			Subject: result.Expression.Range().Ptr(),
		})
	}

	return ret, diags
}

func (c *ComponentInstance) ModuleTreePlan(ctx context.Context) *plans.Plan {
	ret, _ := c.CheckModuleTreePlan(ctx)
	return ret
}

func (c *ComponentInstance) CheckModuleTreePlan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, &c.moduleTreePlan, c.main,
		func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			addr := c.Addr()
			h := hooksFromContext(ctx)
			seq, ctx := hookBegin(ctx, h.BeginComponentInstancePlan, h.ContextAttach, addr)

			decl := c.call.Declaration(ctx)

			// This is our main bridge from the stacks language into the main Terraform
			// module language during the planning phase. We need to ask the main
			// language runtime to plan the module tree associated with this
			// component and return the result.

			moduleTree := c.call.Config(ctx).ModuleTree(ctx)
			if moduleTree == nil {
				// Presumably the configuration is invalid in some way, so
				// we can't create a plan and the relevant diagnostics will
				// get reported when the plan driver visits the ComponentConfig
				// object.
				return nil, diags
			}

			providerSchemas := make(map[addrs.Provider]providers.ProviderSchema)
			for _, sourceAddr := range moduleTree.ProviderTypes() {
				pTy := c.main.ProviderType(ctx, sourceAddr)
				if pTy == nil {
					continue // not our job to report a missing provider type
				}
				schema, err := pTy.Schema(ctx)
				if err != nil {
					// FIXME: it's not technically our job to report a schema
					// fetch failure, but currently there is no single other
					// place that definitely does it, so we'll do it here at
					// the risk of some redundant errors if we end up using
					// the same provider multiple times.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Provider initialization error",
						Detail:   fmt.Sprintf("Failed to fetch the provider schema for %s: %s.", sourceAddr, err),
						Subject:  decl.DeclRange.ToHCL().Ptr(),
					})
					continue // not our job to report a schema fetch failure
				}
				providerSchemas[sourceAddr] = schema
			}
			if diags.HasErrors() {
				return nil, diags
			}

			tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
				Hooks: []terraform.Hook{
					&componentInstanceTerraformHook{
						ctx:   ctx,
						seq:   seq,
						hooks: hooksFromContext(ctx),
						addr:  c.Addr(),
					},
				},
				PreloadedProviderSchemas: providerSchemas,
			})
			if err != nil {
				// Should not get here because we should always pass a valid
				// ContextOpts above.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to instantiate Terraform modules runtime",
					fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
				))
				return nil, diags
			}

			stackPlanOpts := c.main.PlanningOpts()
			inputValues := c.inputValuesForModulesRuntime(ctx, PlanPhase)
			if inputValues == nil {
				// inputValuesForModulesRuntime uses nil (as opposed to a
				// non-nil zerolen map) to represent that the definition of
				// the input variables was so invalid that we cannot do
				// anything with it, in which case we'll just return early
				// and assume the plan walk driver will find the diagnostics
				// via another return path.
				return nil, diags
			}

			providerInstAddrs, valid := c.Providers(ctx, PlanPhase)
			if !valid {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Cannot plan component",
					Detail:   fmt.Sprintf("Cannot generate a plan for %s because its provider configuration assignments are invalid.", c.Addr()),
					Subject:  decl.DeclRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
			providerInsts := make(map[addrs.RootProviderConfig]providers.Interface)
			for calleeAddr, callerAddr := range providerInstAddrs {
				providerInstStack := c.main.Stack(ctx, callerAddr.Stack, PlanPhase)
				if providerInstStack == nil {
					continue
				}
				provider := providerInstStack.Provider(ctx, callerAddr.Item.ProviderConfig)
				if provider == nil {
					continue
				}
				insts := provider.Instances(ctx, PlanPhase)
				if insts == nil {
					// If we get here then we don't yet know which instances
					// this provider has, so we'll be optimistic that it'll
					// show up in a later phase.
					continue
				}
				inst, exists := insts[callerAddr.Item.Key]
				if !exists {
					continue
				}
				providerInsts[calleeAddr] = inst.Client(ctx, PlanPhase)
			}

			// TODO: Should pass in the previous run state once we have a
			// previous stack state to take it from.
			// NOTE: This ComponentInstance type only deals with component
			// instances currently declared in the configuration. See
			// [ComponentInstanceRemoved] for the model of a component instance
			// that existed in the prior state but is not currently declared
			// in the configuration.
			plan, moreDiags := tfCtx.Plan(moduleTree, nil, &terraform.PlanOpts{
				Mode:              stackPlanOpts.PlanningMode,
				SetVariables:      inputValues,
				ExternalProviders: providerInsts,
			})
			diags = diags.Append(moreDiags)

			if plan != nil {
				for _, rsrcChange := range plan.DriftedResources {
					hookMore(ctx, seq, h.ReportResourceInstanceDrift, &hooks.ResourceInstanceChange{
						Addr: stackaddrs.AbsResourceInstance{
							Component: addr,
							Item:      rsrcChange.Addr,
						},
						Change: rsrcChange,
					})
				}
				for _, rsrcChange := range plan.Changes.Resources {
					hookMore(ctx, seq, h.ReportResourceInstancePlanned, &hooks.ResourceInstanceChange{
						Addr: stackaddrs.AbsResourceInstance{
							Component: addr,
							Item:      rsrcChange.Addr,
						},
						Change: rsrcChange,
					})
				}
			}

			if diags.HasErrors() {
				hookMore(ctx, seq, h.ErrorComponentInstancePlan, addr)
			} else {
				hookMore(ctx, seq, h.EndComponentInstancePlan, addr)
			}

			return plan, diags
		},
	)
}

// ApplyModuleTreePlan applies a plan returned by a previous call to
// [ComponentInstance.CheckModuleTreePlan].
//
// Applying a plan often has significant externally-visible side-effects, and
// so this method should be called only once for a given plan. In practice
// we currently ensure that is true by calling it only from the package-level
// [ApplyPlan] function, which arranges for this function to be called
// concurrently with the same method on other component instances and with
// a whole-tree walk to gather up results and diagnostics.
func (c *ComponentInstance) ApplyModuleTreePlan(ctx context.Context, plan *plans.Plan) (*states.State, tfdiags.Diagnostics) {
	panic("unimplemented")
}

// ApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan], or nil if the apply failed and
// so there is no new state to return.
func (c *ComponentInstance) ApplyResultState(ctx context.Context) *states.State {
	ret, _ := c.CheckApplyResultState(ctx)
	return ret
}

// CheckApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan] and diagnostics describing any
// problems encountered when applying it.
func (c *ComponentInstance) CheckApplyResultState(ctx context.Context) (*states.State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	changes := c.main.ApplyChangeResults()
	newState, moreDiags, err := changes.ComponentInstanceResult(ctx, c.Addr())
	diags = diags.Append(moreDiags)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component instance apply not scheduled",
			fmt.Sprintf("Terraform needs the result from applying changes to %s, but that apply was apparently not scheduled to run. This is a bug in Terraform.", c.Addr()),
		))
	}
	return newState, diags
}

func (c *ComponentInstance) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	switch phase {
	case PlanPhase:
		plan := c.ModuleTreePlan(ctx)
		if plan == nil {
			// Planning seems to have failed so we cannot decide a result value yet.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			return cty.DynamicVal
		}

		// During the plan phase we use the planned output changes to construct
		// our value.
		outputChanges := plan.Changes.Outputs
		attrs := make(map[string]cty.Value, len(outputChanges))
		for _, changeSrc := range outputChanges {
			name := changeSrc.Addr.OutputValue.Name
			change, err := changeSrc.Decode()
			if err != nil {
				attrs[name] = cty.DynamicVal
			}
			attrs[name] = change.After
		}
		return cty.ObjectVal(attrs)

	case ApplyPhase:
		newState := c.ApplyResultState(ctx)
		if newState == nil {
			// Applying seems to have failed so we cannot provide a result
			// value, and so we'll return a placeholder to help our caller
			// unwind gracefully with its own placeholder result.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			return cty.DynamicVal
		}

		// During the apply phase we use the root module output values from
		// the new state to construct our value.
		outputVals := newState.RootModule().OutputValues
		attrs := make(map[string]cty.Value, len(outputVals))
		for _, ov := range outputVals {
			name := ov.Addr.OutputValue.Name
			attrs[name] = ov.Value
		}
		return cty.ObjectVal(attrs)

	default:
		// We can't produce a concrete value for any other phase.
		return cty.DynamicVal
	}
}

// ResolveExpressionReference implements ExpressionScope.
func (c *ComponentInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := c.call.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, c.repetition)
}

// PlanChanges implements Plannable by validating that all of the per-instance
// arguments are suitable, and then asking the main Terraform language runtime
// to produce a plan in terms of the component's selected module.
func (c *ComponentInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var changes []stackplan.PlannedChange
	var diags tfdiags.Diagnostics

	hookSingle(ctx, hooksFromContext(ctx).PendingComponentInstancePlan, c.Addr())

	// We must always at least announce that the component instance exists,
	// and that must come before any resource instance changes referring to it.
	changes = append(changes, &stackplan.PlannedChangeComponentInstance{
		Addr: c.Addr(),

		// FIXME: Once we actually have a prior state this should vary
		// depending on whether the same component instance existed in
		// the prior state.
		Action: plans.Create,
	})

	_, moreDiags := c.CheckInputVariableValues(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	_, moreDiags = c.CheckProviders(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	corePlan, moreDiags := c.CheckModuleTreePlan(ctx)
	diags = diags.Append(moreDiags)
	if corePlan != nil {
		for _, rsrcChange := range corePlan.DriftedResources {
			changes = append(changes, &stackplan.PlannedChangeResourceInstanceOutside{
				ComponentInstanceAddr: c.Addr(),
				ChangeSrc:             rsrcChange,
			})
		}
		for _, rsrcChange := range corePlan.Changes.Resources {
			changes = append(changes, &stackplan.PlannedChangeResourceInstancePlanned{
				ComponentInstanceAddr: c.Addr(),
				ChangeSrc:             rsrcChange,
			})
		}
	}

	return changes, diags
}

// CheckApply implements ApplyChecker.
func (c *ComponentInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var changes []stackstate.AppliedChange
	var diags tfdiags.Diagnostics

	// FIXME: We need to report AppliedChange objects for the component
	// instance itself and each of the affected resource instances inside it.
	// For now we're only reporting diagnostics as an initial stub.

	_, moreDiags := c.CheckInputVariableValues(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	_, moreDiags = c.CheckProviders(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	_, moreDiags = c.CheckApplyResultState(ctx)
	diags = diags.Append(moreDiags)

	return changes, diags
}

func (c *ComponentInstance) tracingName() string {
	return c.Addr().String()
}
