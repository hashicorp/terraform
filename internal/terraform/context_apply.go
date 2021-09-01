package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Apply performs the actions described by the given Plan object and returns
// the resulting updated state.
//
// The given configuration *must* be the same configuration that was passed
// earlier to Context.Plan in order to create this plan.
//
// Even if the returned diagnostics contains errors, Apply always returns the
// resulting state which is likely to have been partially-updated.
func (c *Context) Apply(plan *plans.Plan, config *configs.Config) (*states.State, tfdiags.Diagnostics) {
	defer c.acquireRun("apply")()
	var diags tfdiags.Diagnostics

	log.Printf("[DEBUG] Building and walking apply graph for %s plan", plan.UIMode)

	graph, operation, moreDiags := c.applyGraph(plan, config, true)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	variables := InputValues{}
	for name, dyVal := range plan.VariableValues {
		val, err := dyVal.Decode(cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid variable value in plan",
				fmt.Sprintf("Invalid value for variable %q recorded in plan file: %s.", name, err),
			))
			continue
		}

		variables[name] = &InputValue{
			Value:      val,
			SourceType: ValueFromPlan,
		}
	}

	workingState := plan.PriorState.DeepCopy()
	walker, walkDiags := c.walk(graph, operation, &graphWalkOpts{
		Config:             config,
		InputState:         workingState,
		Changes:            plan.Changes,
		RootVariableValues: variables,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	newState := walker.State.Close()
	if plan.UIMode == plans.DestroyMode && !diags.HasErrors() {
		// NOTE: This is a vestigial violation of the rule that we mustn't
		// use plan.UIMode to affect apply-time behavior.
		// We ideally ought to just call newState.PruneResourceHusks
		// unconditionally here, but we historically didn't and haven't yet
		// verified that it'd be safe to do so.
		newState.PruneResourceHusks()
	}

	if len(plan.TargetAddrs) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Applied changes may be incomplete",
			`The plan was created with the -target option in effect, so some changes requested in the configuration may have been ignored and the output values may not be fully updated. Run the following command to verify that no other changes are pending:
    terraform plan
	
Note that the -target option is not suitable for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
		))
	}

	return newState, diags
}

func (c *Context) applyGraph(plan *plans.Plan, config *configs.Config, validate bool) (*Graph, walkOperation, tfdiags.Diagnostics) {
	graph, diags := (&ApplyGraphBuilder{
		Config:       config,
		Changes:      plan.Changes,
		State:        plan.PriorState,
		Plugins:      c.plugins,
		Targets:      plan.TargetAddrs,
		ForceReplace: plan.ForceReplaceAddrs,
		Validate:     validate,
	}).Build(addrs.RootModuleInstance)

	operation := walkApply
	if plan.UIMode == plans.DestroyMode {
		// NOTE: This is a vestigial violation of the rule that we mustn't
		// use plan.UIMode to affect apply-time behavior. It's a design error
		// if anything downstream switches behavior when operation is set
		// to walkDestroy, but we've not yet fully audited that.
		// TODO: Audit that and remove walkDestroy as an operation mode.
		operation = walkDestroy
	}

	return graph, operation, diags
}

// ApplyGraphForUI is a last vestage of graphs in the public interface of
// Context (as opposed to graphs as an implementation detail) intended only for
// use by the "terraform graph" command when asked to render an apply-time
// graph.
//
// The result of this is intended only for rendering ot the user as a dot
// graph, and so may change in future in order to make the result more useful
// in that context, even if drifts away from the physical graph that Terraform
// Core currently uses as an implementation detail of planning.
func (c *Context) ApplyGraphForUI(plan *plans.Plan, config *configs.Config) (*Graph, tfdiags.Diagnostics) {
	// For now though, this really is just the internal graph, confusing
	// implementation details and all.

	var diags tfdiags.Diagnostics

	graph, _, moreDiags := c.applyGraph(plan, config, false)
	diags = diags.Append(moreDiags)
	return graph, diags
}
