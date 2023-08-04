package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestContext wraps a Context, and adds in direct values for the current state,
// most recent plan, and configuration.
//
// This combination allows functions called on the TestContext to create a
// complete scope to evaluate test assertions.
type TestContext struct {
	*Context

	Config    *configs.Config
	State     *states.State
	Plan      *plans.Plan
	Variables InputValues
}

// TestContext creates a TestContext structure that can evaluate test assertions
// against the provided state and plan.
func (c *Context) TestContext(config *configs.Config, state *states.State, plan *plans.Plan, variables InputValues) *TestContext {
	return &TestContext{
		Context:   c,
		Config:    config,
		State:     state,
		Plan:      plan,
		Variables: variables,
	}
}

// EvaluateAgainstState processes the assertions inside the provided
// configs.TestRun against the embedded state.
//
// The provided plan is import as it is needed to evaluate the `plantimestamp`
// function, but no data or changes from the embedded plan is referenced in
// this function.
func (ctx *TestContext) EvaluateAgainstState(run *moduletest.Run) {
	defer ctx.acquireRun("evaluate")()
	ctx.evaluate(ctx.State.SyncWrapper(), plans.NewChanges().SyncWrapper(), run, walkApply)
}

// EvaluateAgainstPlan processes the assertions inside the provided
// configs.TestRun against the embedded plan and state.
func (ctx *TestContext) EvaluateAgainstPlan(run *moduletest.Run) {
	defer ctx.acquireRun("evaluate")()
	ctx.evaluate(ctx.State.SyncWrapper(), ctx.Plan.Changes.SyncWrapper(), run, walkPlan)
}

func (ctx *TestContext) evaluate(state *states.SyncState, changes *plans.ChangesSync, run *moduletest.Run, operation walkOperation) {
	data := &evaluationStateData{
		Evaluator: &Evaluator{
			Operation: operation,
			Meta:      ctx.meta,
			Config:    ctx.Config,
			Plugins:   ctx.plugins,
			State:     state,
			Changes:   changes,
			VariableValues: func() map[string]map[string]cty.Value {
				variables := map[string]map[string]cty.Value{
					addrs.RootModule.String(): make(map[string]cty.Value),
				}
				for name, variable := range ctx.Variables {
					variables[addrs.RootModule.String()][name] = variable.Value
				}
				return variables
			}(),
			VariableValuesLock: new(sync.Mutex),
			PlanTimestamp:      ctx.Plan.Timestamp,
		},
		ModulePath:      nil, // nil for the root module
		InstanceKeyData: EvalDataForNoInstanceKey,
		Operation:       operation,
	}

	scope := &lang.Scope{
		Data:          data,
		BaseDir:       ".",
		PureOnly:      operation != walkApply,
		PlanTimestamp: ctx.Plan.Timestamp,
	}

	// We're going to assume the run has passed, and then if anything fails this
	// value will be updated.
	run.Status = run.Status.Merge(moduletest.Pass)

	// Now validate all the assertions within this run block.
	for _, rule := range run.Config.CheckRules {
		var diags tfdiags.Diagnostics

		refs, moreDiags := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.Condition)
		diags = diags.Append(moreDiags)
		moreRefs, moreDiags := lang.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.ErrorMessage)
		diags = diags.Append(moreDiags)
		refs = append(refs, moreRefs...)

		hclCtx, moreDiags := scope.EvalContext(refs)
		diags = diags.Append(moreDiags)

		errorMessage, moreDiags := evalCheckErrorMessage(rule.ErrorMessage, hclCtx)
		diags = diags.Append(moreDiags)

		runVal, hclDiags := rule.Condition.Value(hclCtx)
		diags = diags.Append(hclDiags)

		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = run.Status.Merge(moduletest.Error)
			continue
		}

		if runVal.IsNull() {
			run.Status = run.Status.Merge(moduletest.Error)
			run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      "Condition expression must return either true or false, not null.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		if !runVal.IsKnown() {
			run.Status = run.Status.Merge(moduletest.Error)
			run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Unknown condition run",
				Detail:      "Condition expression could not be evaluated at this time.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		var err error
		if runVal, err = convert.Convert(runVal, cty.Bool); err != nil {
			run.Status = run.Status.Merge(moduletest.Error)
			run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      fmt.Sprintf("Invalid condition run value: %s.", tfdiags.FormatError(err)),
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		if runVal.False() {
			run.Status = run.Status.Merge(moduletest.Fail)
			run.Diagnostics = run.Diagnostics.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Test assertion failed",
				Detail:      errorMessage,
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		}
	}
}
