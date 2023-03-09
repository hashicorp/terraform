// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/namedvals"
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

func (ctx *TestContext) evaluationStateData(alternateStates map[string]*evaluationStateData) (*evaluationStateData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var operation walkOperation
	switch ctx.Run.Config.Command {
	case configs.PlanTestCommand:
		operation = walkPlan
	case configs.ApplyTestCommand:
		operation = walkApply
	default:
		panic(fmt.Errorf("unrecognized TestCommand: %q", ctx.Run.Config.Command))
	}

	variableValues := make(map[string]map[string]cty.Value)
	variableValues[addrs.RootModule.String()] = make(map[string]cty.Value)
	for name, variable := range ctx.Variables {
		if config, exists := ctx.Config.Module.Variables[name]; exists {
			var variableDiags tfdiags.Diagnostics
			variableValues[addrs.RootModule.String()][name], variableDiags = prepareFinalInputVariableValue(addrs.RootModuleInstance.InputVariable(name), variable, config)
			diags = diags.Append(variableDiags)
		} else {
			variableValues[addrs.RootModule.String()][name] = variable.Value
		}
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
			NamedValues:     namedvals.NewState(),
			PlanTimestamp:   ctx.Plan.Timestamp,
		},
		ModulePath:      nil, // nil for the root module
		InstanceKeyData: EvalDataForNoInstanceKey,
		Operation:       operation,
	}, diags
}

// Evaluate processes the assertions inside the provided configs.TestRun against
// the embedded state.
func (ctx *TestContext) Evaluate(priorContexts map[string]*TestContext) {
	run := ctx.Run

	var dataDiags tfdiags.Diagnostics
	alternateStates := make(map[string]*evaluationStateData)
	for name, priorContext := range priorContexts {
		if priorContext == nil {
			// Skip contexts that haven't been executed yet.
			continue
		}

		var moreDiags tfdiags.Diagnostics
		alternateStates[name], moreDiags = priorContext.evaluationStateData(nil)
		dataDiags = dataDiags.Append(moreDiags)
	}

	data, moreDiags := ctx.evaluationStateData(alternateStates)
	scope := &lang.Scope{
		Data:          data,
		BaseDir:       ".",
		PureOnly:      data.Operation != walkApply,
		PlanTimestamp: ctx.Plan.Timestamp,
	}
	dataDiags = dataDiags.Append(moreDiags)

	run.Diagnostics = run.Diagnostics.Append(dataDiags)
	if dataDiags.HasErrors() {
		// Fail early if we couldn't adequately build the evaluation state data.
		run.Status = run.Status.Merge(moduletest.Error)
		return
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
