// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/mnptu/internal/addrs"
	"github.com/hashicorp/mnptu/internal/configs"
	"github.com/hashicorp/mnptu/internal/lang"
	"github.com/hashicorp/mnptu/internal/moduletest"
	"github.com/hashicorp/mnptu/internal/plans"
	"github.com/hashicorp/mnptu/internal/states"
	"github.com/hashicorp/mnptu/internal/tfdiags"
)

// TestContext wraps a Context, and adds in direct values for the current state,
// most recent plan, and configuration.
//
// This combination allows functions called on the TestContext to create a
// complete scope to evaluate test assertions.
type TestContext struct {
	*Context

	Run       *moduletest.Run
	Config    *configs.Config
	State     *states.State
	Plan      *plans.Plan
	Variables InputValues
}

// TestContext creates a TestContext structure that can evaluate test assertions
// against the provided state and plan.
func (c *Context) TestContext(run *moduletest.Run, config *configs.Config, state *states.State, plan *plans.Plan, variables InputValues) *TestContext {
	return &TestContext{
		Context:   c,
		Run:       run,
		Config:    config,
		State:     state,
		Plan:      plan,
		Variables: variables,
	}
}

func (ctx *TestContext) evaluationStateData(alternateStates map[string]*evaluationStateData) *evaluationStateData {

	var operation walkOperation
	switch ctx.Run.Config.Command {
	case configs.PlanTestCommand:
		operation = walkPlan
	case configs.ApplyTestCommand:
		operation = walkApply
	default:
		panic(fmt.Errorf("unrecognized TestCommand: %q", ctx.Run.Config.Command))
	}

	return &evaluationStateData{
		Evaluator: &Evaluator{
			Operation:       operation,
			Meta:            ctx.meta,
			Config:          ctx.Config,
			Plugins:         ctx.plugins,
			State:           ctx.State.SyncWrapper(),
			Changes:         ctx.Plan.Changes.SyncWrapper(),
			AlternateStates: alternateStates,
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
}

// Evaluate processes the assertions inside the provided configs.TestRun against
// the embedded state.
func (ctx *TestContext) Evaluate(priorContexts map[string]*TestContext) {

	alternateStates := make(map[string]*evaluationStateData)
	for name, priorContext := range priorContexts {
		alternateStates[name] = priorContext.evaluationStateData(nil)
	}

	data := ctx.evaluationStateData(alternateStates)
	scope := &lang.Scope{
		Data:          data,
		BaseDir:       ".",
		PureOnly:      data.Operation != walkApply,
		PlanTimestamp: ctx.Plan.Timestamp,
	}

	// We're going to assume the run has passed, and then if anything fails this
	// value will be updated.
	run := ctx.Run
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
				Summary:     "Unknown condition value",
				Detail:      "Condition expression could not be evaluated at this time. This means you have executed a `run` block with `command = plan` and one of the values your condition depended on is not known until after the plan has been applied. Either remove this value from your condition, or execute an `apply` command from this `run` block.",
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

		// If the runVal refers to any sensitive values, then we'll have a
		// sensitive mark on the resulting value.
		runVal, _ = runVal.Unmark()

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
