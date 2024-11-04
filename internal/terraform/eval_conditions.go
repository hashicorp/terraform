// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evalCheckRules ensures that all of the given check rules pass against
// the given HCL evaluation context.
//
// If any check rules produce an unknown result then they will be silently
// ignored on the assumption that the same checks will be run again later
// with fewer unknown values in the EvalContext.
//
// If any of the rules do not pass, the returned diagnostics will contain
// errors. Otherwise, it will either be empty or contain only warnings.
func evalCheckRules(typ addrs.CheckRuleType, rules []*configs.CheckRule, ctx EvalContext, self addrs.Checkable, keyData instances.RepetitionData, diagSeverity tfdiags.Severity) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	checkState := ctx.Checks()
	if !checkState.ConfigHasChecks(self.ConfigCheckable()) {
		// We have nothing to do if this object doesn't have any checks,
		// but the "rules" slice should agree that we don't.
		if ct := len(rules); ct != 0 {
			panic(fmt.Sprintf("check state says that %s should have no rules, but it has %d", self, ct))
		}
		return diags
	}

	if len(rules) == 0 {
		// Nothing to do
		return nil
	}

	severity := diagSeverity.ToHCL()

	for i, rule := range rules {
		result, ruleDiags := evalCheckRule(addrs.NewCheckRule(self, typ, i), rule, ctx, keyData, severity)
		diags = diags.Append(ruleDiags)

		log.Printf("[TRACE] evalCheckRules: %s status is now %s", self, result.Status)
		if result.Status == checks.StatusFail {
			checkState.ReportCheckFailure(self, typ, i, result.FailureMessage)
		} else {
			checkState.ReportCheckResult(self, typ, i, result.Status)
		}
	}

	return diags
}

type checkResult struct {
	Status         checks.Status
	FailureMessage string
}

func validateCheckRule(addr addrs.CheckRule, rule *configs.CheckRule, ctx EvalContext, keyData instances.RepetitionData) (string, *hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	refs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, rule.Condition)
	diags = diags.Append(moreDiags)
	moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, rule.ErrorMessage)
	diags = diags.Append(moreDiags)
	refs = append(refs, moreRefs...)

	var selfReference, sourceReference addrs.Referenceable
	switch addr.Type {
	case addrs.ResourcePostcondition:
		switch s := addr.Container.(type) {
		case addrs.AbsResourceInstance:
			// Only resource postconditions can refer to self
			selfReference = s.Resource
		default:
			panic(fmt.Sprintf("Invalid self reference type %t", addr.Container))
		}
	case addrs.CheckAssertion:
		switch s := addr.Container.(type) {
		case addrs.AbsCheck:
			// Only check blocks have scoped resources so need to specify their
			// source.
			sourceReference = s.Check
		default:
			panic(fmt.Sprintf("Invalid source reference type %t", addr.Container))
		}
	}
	scope := ctx.EvaluationScope(selfReference, sourceReference, keyData)

	hclCtx, moreDiags := scope.EvalContext(refs)
	diags = diags.Append(moreDiags)

	errorMessage, moreDiags := lang.EvalCheckErrorMessage(rule.ErrorMessage, hclCtx, &addr)
	diags = diags.Append(moreDiags)

	return errorMessage, hclCtx, diags
}

func evalCheckRule(addr addrs.CheckRule, rule *configs.CheckRule, ctx EvalContext, keyData instances.RepetitionData, severity hcl.DiagnosticSeverity) (checkResult, tfdiags.Diagnostics) {
	// NOTE: Intentionally not passing the caller's selected severity in here,
	// because this reports errors in the configuration itself, not the failure
	// of an otherwise-valid condition.
	errorMessage, hclCtx, diags := validateCheckRule(addr, rule, ctx, keyData)

	const errInvalidCondition = "Invalid condition result"

	resultVal, hclDiags := rule.Condition.Value(hclCtx)
	diags = diags.Append(hclDiags)

	if diags.HasErrors() {
		log.Printf("[TRACE] evalCheckRule: %s: %s", addr.Type, diags.Err().Error())
		return checkResult{Status: checks.StatusError}, diags
	}

	if !resultVal.IsKnown() {

		// Check assertions warn if a status is unknown.
		if addr.Type == addrs.CheckAssertion {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagWarning,
				Summary:     fmt.Sprintf("%s known after apply", addr.Type.Description()),
				Detail:      "The condition could not be evaluated at this time, a result will be known when this plan is applied.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
				Extra: &addrs.CheckRuleDiagnosticExtra{
					CheckRule: addr,
				},
			})
		}

		// We'll wait until we've learned more, then.
		return checkResult{Status: checks.StatusUnknown}, diags
	}
	if resultVal.IsNull() {
		// NOTE: Intentionally not passing the caller's selected severity in here,
		// because this reports errors in the configuration itself, not the failure
		// of an otherwise-valid condition.
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      "Condition expression must return either true or false, not null.",
			Subject:     rule.Condition.Range().Ptr(),
			Expression:  rule.Condition,
			EvalContext: hclCtx,
		})
		return checkResult{Status: checks.StatusError}, diags
	}
	var err error
	resultVal, err = convert.Convert(resultVal, cty.Bool)
	if err != nil {
		// NOTE: Intentionally not passing the caller's selected severity in here,
		// because this reports errors in the configuration itself, not the failure
		// of an otherwise-valid condition.
		detail := fmt.Sprintf("Invalid condition result value: %s.", tfdiags.FormatError(err))
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      detail,
			Subject:     rule.Condition.Range().Ptr(),
			Expression:  rule.Condition,
			EvalContext: hclCtx,
		})
		return checkResult{Status: checks.StatusError}, diags
	}

	// The condition result may be marked if the expression refers to a
	// sensitive value.
	resultVal, _ = resultVal.Unmark()

	status := checks.StatusForCtyValue(resultVal)

	if status != checks.StatusFail {
		return checkResult{Status: status}, diags
	}

	errorMessageForDiags := errorMessage
	if errorMessageForDiags == "" {
		errorMessageForDiags = "This check failed, but has an invalid error message as described in the other accompanying messages."
	}
	diags = diags.Append(&hcl.Diagnostic{
		// The caller gets to choose the severity of this one, because we
		// treat condition failures as warnings in the presence of
		// certain special planning options.
		Severity:    severity,
		Summary:     fmt.Sprintf("%s failed", addr.Type.Description()),
		Detail:      errorMessageForDiags,
		Subject:     rule.Condition.Range().Ptr(),
		Expression:  rule.Condition,
		EvalContext: hclCtx,
		Extra: &addrs.CheckRuleDiagnosticExtra{
			CheckRule: addr,
		},
	})

	return checkResult{
		Status:         status,
		FailureMessage: errorMessage,
	}, diags
}
