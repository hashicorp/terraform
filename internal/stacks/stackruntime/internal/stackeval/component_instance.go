package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	tfProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
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
	}

	v = defs.Apply(v)
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

func (c *ComponentInstance) ModuleTreePlan(ctx context.Context) *plans.Plan {
	ret, _ := c.CheckModuleTreePlan(ctx)
	return ret
}

func (c *ComponentInstance) CheckModuleTreePlan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, &c.moduleTreePlan, c.main,
		func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

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

			// FIXME: This is just a temporary stub with various things
			// hard-coded for now. In a real implementation we'd need to
			// be passed in provider factories from outside the runtime,
			// and populate various other things to make this actually
			// work like existing Terraform modules expect.
			tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.MustParseProviderSourceString("terraform.io/builtin/terraform"): func() (providers.Interface, error) {
						return tfProvider.NewProvider(), nil
					},
				},
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

			// TODO: Should pass in the previous run state once we have a
			// previous stack state to take it from.
			// NOTE: This ComponentInstance type only deals with component
			// instances currently declared in the configuration. See
			// [ComponentInstanceRemoved] for the model of a component instance
			// that existed in the prior state but is not currently declared
			// in the configuration.
			plan, moreDiags := tfCtx.Plan(moduleTree, nil, &terraform.PlanOpts{
				Mode:         stackPlanOpts.PlanningMode,
				SetVariables: inputValues,
			})
			diags = diags.Append(moreDiags)
			return plan, diags
		},
	)
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

	_, moreDiags := c.CheckInputVariableValues(ctx, PlanPhase)
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

func (c *ComponentInstance) tracingName() string {
	return c.Addr().String()
}
