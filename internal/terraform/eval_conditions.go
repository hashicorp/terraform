package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
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
func evalCheckRules(typ addrs.CheckType, rules []*configs.CheckRule, ctx EvalContext, self addrs.Checkable, keyData instances.RepetitionData, diagSeverity tfdiags.Severity) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(rules) == 0 {
		// Nothing to do
		return nil
	}

	severity := diagSeverity.ToHCL()

	for i, rule := range rules {
		checkAddr := self.Check(typ, i)

		conditionResult, ruleDiags := evalCheckRule(typ, rule, ctx, self, keyData, severity)
		diags = diags.Append(ruleDiags)
		ctx.Conditions().SetResult(checkAddr, conditionResult)
	}

	return diags
}

func evalCheckRule(typ addrs.CheckType, rule *configs.CheckRule, ctx EvalContext, self addrs.Checkable, keyData instances.RepetitionData, severity hcl.DiagnosticSeverity) (*plans.ConditionResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	const errInvalidCondition = "Invalid condition result"

	refs, moreDiags := lang.ReferencesInExpr(rule.Condition)
	diags = diags.Append(moreDiags)
	moreRefs, moreDiags := lang.ReferencesInExpr(rule.ErrorMessage)
	diags = diags.Append(moreDiags)
	refs = append(refs, moreRefs...)

	conditionResult := &plans.ConditionResult{
		Address: self,
		Result:  cty.UnknownVal(cty.Bool),
		Type:    typ,
	}

	var selfReference addrs.Referenceable
	// Only resource postconditions can refer to self
	if typ == addrs.ResourcePostcondition {
		switch s := self.(type) {
		case addrs.AbsResourceInstance:
			selfReference = s.Resource
		default:
			panic(fmt.Sprintf("Invalid self reference type %t", self))
		}
	}
	scope := ctx.EvaluationScope(selfReference, keyData)

	hclCtx, moreDiags := scope.EvalContext(refs)
	diags = diags.Append(moreDiags)

	result, hclDiags := rule.Condition.Value(hclCtx)
	diags = diags.Append(hclDiags)

	errorValue, errorDiags := rule.ErrorMessage.Value(hclCtx)
	diags = diags.Append(errorDiags)

	if diags.HasErrors() {
		log.Printf("[TRACE] evalCheckRule: %s: %s", typ, diags.Err().Error())
	}

	if !result.IsKnown() {
		// We'll wait until we've learned more, then.
		return conditionResult, diags
	}

	if result.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    severity,
			Summary:     errInvalidCondition,
			Detail:      "Condition expression must return either true or false, not null.",
			Subject:     rule.Condition.Range().Ptr(),
			Expression:  rule.Condition,
			EvalContext: hclCtx,
		})
		conditionResult.Result = cty.False
		conditionResult.ErrorMessage = "Condition expression must return either true or false, not null."
		return conditionResult, diags
	}
	var err error
	result, err = convert.Convert(result, cty.Bool)
	if err != nil {
		detail := fmt.Sprintf("Invalid condition result value: %s.", tfdiags.FormatError(err))
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    severity,
			Summary:     errInvalidCondition,
			Detail:      detail,
			Subject:     rule.Condition.Range().Ptr(),
			Expression:  rule.Condition,
			EvalContext: hclCtx,
		})
		conditionResult.Result = cty.False
		conditionResult.ErrorMessage = detail
		return conditionResult, diags
	}

	// The condition result may be marked if the expression refers to a
	// sensitive value.
	result, _ = result.Unmark()
	conditionResult.Result = result

	if result.True() {
		return conditionResult, diags
	}

	var errorMessage string
	if !errorDiags.HasErrors() && errorValue.IsKnown() && !errorValue.IsNull() {
		var err error
		errorValue, err = convert.Convert(errorValue, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    severity,
				Summary:     "Invalid error message",
				Detail:      fmt.Sprintf("Unsuitable value for error message: %s.", tfdiags.FormatError(err)),
				Subject:     rule.ErrorMessage.Range().Ptr(),
				Expression:  rule.ErrorMessage,
				EvalContext: hclCtx,
			})
		} else {
			if marks.Has(errorValue, marks.Sensitive) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: severity,

					Summary: "Error message refers to sensitive values",
					Detail: `The error expression used to explain this condition refers to sensitive values. Terraform will not display the resulting message.

You can correct this by removing references to sensitive values, or by carefully using the nonsensitive() function if the expression will not reveal the sensitive data.`,

					Subject:     rule.ErrorMessage.Range().Ptr(),
					Expression:  rule.ErrorMessage,
					EvalContext: hclCtx,
				})
				errorMessage = "The error message included a sensitive value, so it will not be displayed."
			} else {
				errorMessage = strings.TrimSpace(errorValue.AsString())
			}
		}
	}
	if errorMessage == "" {
		errorMessage = "Failed to evaluate condition error message."
	}
	diags = diags.Append(&hcl.Diagnostic{
		Severity:    severity,
		Summary:     fmt.Sprintf("%s failed", typ.Description()),
		Detail:      errorMessage,
		Subject:     rule.Condition.Range().Ptr(),
		Expression:  rule.Condition,
		EvalContext: hclCtx,
	})
	conditionResult.ErrorMessage = errorMessage
	return conditionResult, diags
}
