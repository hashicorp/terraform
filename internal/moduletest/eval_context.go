// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EvalContext is a container for context relating to the evaluation of a
// particular test case, which means a specific "run" block in a .tftest.hcl
// file.
type EvalContext struct {
	run       *Run
	module    *configs.Module
	exprScope *lang.Scope
}

// NewEvalContext constructs a new test run evaluation context based on the
// definition of the run itself and on the results of the action the run
// block described.
//
// priorOutputs describes the output values from earlier run blocks, which
// should typically be populated from the second return value from calling
// [EvalContext.Evaluate] on each earlier blocks' [EvalContext].
//
// extraVariableVals, if provided, overlays the input variables that are
// already available in resultScope in case there are additional input
// variables that were defined only for use in the test suite. Any variable
// not defined in extraVariableVals will be evaluated through resultScope
// instead.
func NewEvalContext(run *Run, module *configs.Module, resultScope *lang.Scope, extraVariableVals terraform.InputValues, priorOutputs map[addrs.Run]cty.Value) *EvalContext {
	// We need a derived evaluation scope that also supports referring to
	// the prior run output values using the "run.NAME" syntax.
	evalData := &evaluationData{
		module:    module,
		current:   resultScope.Data,
		extraVars: extraVariableVals,
		priorVals: priorOutputs,
	}
	runScope := &lang.Scope{
		Data:          evalData,
		ParseRef:      addrs.ParseRefFromTestingScope,
		SourceAddr:    resultScope.SourceAddr,
		BaseDir:       resultScope.BaseDir,
		PureOnly:      resultScope.PureOnly,
		PlanTimestamp: resultScope.PlanTimestamp,
		ExternalFuncs: resultScope.ExternalFuncs,
	}
	return &EvalContext{
		run:       run,
		module:    module,
		exprScope: runScope,
	}
}

// Evaluate processes the assertions inside the provided configs.TestRun against
// the run results, returning a status, an object value representing the output
// values from the module under test, and diagnostics describing any problems.
func (ec *EvalContext) Evaluate() (Status, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	run := ec.run
	scope := ec.exprScope

	log.Printf("[TRACE] EvalContext.Evaluate for %s", ec.run.Addr())

	// We're going to assume the run has passed, and then if anything fails this
	// value will be updated.
	status := run.Status.Merge(Pass)

	// Now validate all the assertions within this run block.
	for i, rule := range run.Config.CheckRules {
		var ruleDiags tfdiags.Diagnostics

		refs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.Condition)
		ruleDiags = ruleDiags.Append(moreDiags)
		moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.ErrorMessage)
		ruleDiags = ruleDiags.Append(moreDiags)
		refs = append(refs, moreRefs...)

		hclCtx, moreDiags := scope.EvalContext(refs)
		ruleDiags = ruleDiags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// if we can't evaluate the context properly, we can't evaulate the rule
			// we add the diagnostics to the main diags and continue to the next rule
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, could not evalaute the context, so cannot evaluate it", i, ec.run.Addr())
			status = status.Merge(Error)
			diags = diags.Append(ruleDiags)
			continue
		}

		errorMessage, moreDiags := lang.EvalCheckErrorMessage(rule.ErrorMessage, hclCtx)
		ruleDiags = ruleDiags.Append(moreDiags)

		runVal, hclDiags := rule.Condition.Value(hclCtx)
		ruleDiags = ruleDiags.Append(hclDiags)

		diags = diags.Append(ruleDiags)
		if ruleDiags.HasErrors() {
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, so cannot evaluate it", i, ec.run.Addr())
			status = status.Merge(Error)
			continue
		}

		if runVal.IsNull() {
			status = status.Merge(Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      "Condition expression must return either true or false, not null.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has null condition result", i, ec.run.Addr())
			continue
		}

		if !runVal.IsKnown() {
			status = status.Merge(Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Unknown condition value",
				Detail:      "Condition expression could not be evaluated at this time. This means you have executed a `run` block with `command = plan` and one of the values your condition depended on is not known until after the plan has been applied. Either remove this value from your condition, or execute an `apply` command from this `run` block.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has unknown condition result", i, ec.run.Addr())
			continue
		}

		var err error
		if runVal, err = convert.Convert(runVal, cty.Bool); err != nil {
			status = status.Merge(Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      fmt.Sprintf("Invalid condition run value: %s.", tfdiags.FormatError(err)),
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has non-boolean condition result", i, ec.run.Addr())
			continue
		}

		// If the runVal refers to any sensitive values, then we'll have a
		// sensitive mark on the resulting value.
		runVal, _ = runVal.Unmark()

		if runVal.False() {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion failed for %s assertion %d", ec.run.Addr(), i)
			status = status.Merge(Fail)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Test assertion failed",
				Detail:      errorMessage,
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		} else {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion succeeded for %s assertion %d", ec.run.Addr(), i)
		}
	}

	// Our result includes an object representing all of the output values
	// from the module we've just tested, which will then be available in
	// any subsequent test cases in the same test suite.
	outputVals := make(map[string]cty.Value, len(ec.module.Outputs))
	runRng := tfdiags.SourceRangeFromHCL(ec.run.Config.DeclRange)
	for _, oc := range ec.module.Outputs {
		addr := oc.Addr()
		v, moreDiags := ec.exprScope.Data.GetOutput(addr, runRng)
		diags = diags.Append(moreDiags)
		if v == cty.NilVal {
			v = cty.NullVal(cty.DynamicPseudoType)
		}
		outputVals[addr.Name] = v
	}

	return status, cty.ObjectVal(outputVals), diags
}

// evaluationData augments an underlying lang.Data -- presumably resulting
// from a terraform.Context.PlanAndEval or terraform.Context.ApplyAndEval call --
// with results from prior runs that should therefore be available when
// evaluating expressions written inside a "run" block.
type evaluationData struct {
	module    *configs.Module
	current   lang.Data
	extraVars terraform.InputValues
	priorVals map[addrs.Run]cty.Value
}

var _ lang.Data = (*evaluationData)(nil)

// GetCheckBlock implements lang.Data.
func (d *evaluationData) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetCheckBlock(addr, rng)
}

// GetCountAttr implements lang.Data.
func (d *evaluationData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetCountAttr(addr, rng)
}

// GetForEachAttr implements lang.Data.
func (d *evaluationData) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetForEachAttr(addr, rng)
}

// GetInputVariable implements lang.Data.
func (d *evaluationData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	if extra, exists := d.extraVars[addr.Name]; exists {
		return extra.Value, nil
	}
	return d.current.GetInputVariable(addr, rng)
}

// GetLocalValue implements lang.Data.
func (d *evaluationData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetLocalValue(addr, rng)
}

// GetModule implements lang.Data.
func (d *evaluationData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetModule(addr, rng)
}

// GetOutput implements lang.Data.
func (d *evaluationData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetOutput(addr, rng)
}

// GetPathAttr implements lang.Data.
func (d *evaluationData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetPathAttr(addr, rng)
}

// GetResource implements lang.Data.
func (d *evaluationData) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetResource(addr, rng)
}

// GetRunBlock implements lang.Data.
func (d *evaluationData) GetRunBlock(addr addrs.Run, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret, exists := d.priorVals[addr]
	if !exists {
		ret = cty.DynamicVal
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared run block",
			Detail:   fmt.Sprintf("There is no run %q declared in this test suite.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}
	if ret == cty.NilVal {
		// An explicit nil value indicates that the block was declared but
		// hasn't yet been visited.
		ret = cty.DynamicVal
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unevaluated run block",
			Detail:   fmt.Sprintf("The run %q block has not yet been evaluated, so its results are not available here.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}
	return ret, diags
}

// GetTerraformAttr implements lang.Data.
func (d *evaluationData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetTerraformAttr(addr, rng)
}

// StaticValidateReferences implements lang.Data.
func (d *evaluationData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	// We only handle addrs.Run directly here, with everything else delegated
	// to the underlying Data object to deal with.
	var diags tfdiags.Diagnostics
	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.Run:
			diags = diags.Append(d.staticValidateRunRef(ref))
		default:
			diags = diags.Append(d.current.StaticValidateReferences([]*addrs.Reference{ref}, self, source))
		}
	}
	return diags
}

func (d *evaluationData) staticValidateRunRef(ref *addrs.Reference) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	addr := ref.Subject.(addrs.Run)
	_, exists := d.priorVals[addr]
	if !exists {
		var suggestions []string
		for altAddr := range d.priorVals {
			suggestions = append(suggestions, altAddr.Name)
		}
		sort.Strings(suggestions)
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		// A totally absent priorVals means that there is no run block with
		// the given name at all. If it was declared but hasn't yet been
		// evaluated then it would have an entry set to cty.NilVal.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared run block",
			Detail:   fmt.Sprintf("There is no run %q declared in this test suite.%s", addr.Name, suggestion),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
	}

	return diags
}
