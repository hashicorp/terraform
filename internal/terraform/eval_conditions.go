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
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type checkType int

const (
	checkInvalid               checkType = 0
	checkResourcePrecondition  checkType = 1
	checkResourcePostcondition checkType = 2
	checkOutputPrecondition    checkType = 3
)

func (c checkType) FailureSummary() string {
	switch c {
	case checkResourcePrecondition:
		return "Resource precondition failed"
	case checkResourcePostcondition:
		return "Resource postcondition failed"
	case checkOutputPrecondition:
		return "Module output value precondition failed"
	default:
		// This should not happen
		return "Failed condition for invalid check type"
	}
}

// evalCheckRules ensures that all of the given check rules pass against
// the given HCL evaluation context.
//
// If any check rules produce an unknown result then they will be silently
// ignored on the assumption that the same checks will be run again later
// with fewer unknown values in the EvalContext.
//
// If any of the rules do not pass, the returned diagnostics will contain
// errors. Otherwise, it will either be empty or contain only warnings.
func evalCheckRules(typ checkType, rules []*configs.CheckRule, ctx EvalContext, self addrs.Referenceable, keyData instances.RepetitionData, diagSeverity tfdiags.Severity) (diags tfdiags.Diagnostics) {
	if len(rules) == 0 {
		// Nothing to do
		return nil
	}

	severity := diagSeverity.ToHCL()

	for _, rule := range rules {
		const errInvalidCondition = "Invalid condition result"
		var ruleDiags tfdiags.Diagnostics

		refs, moreDiags := lang.ReferencesInExpr(rule.Condition)
		ruleDiags = ruleDiags.Append(moreDiags)
		moreRefs, moreDiags := lang.ReferencesInExpr(rule.ErrorMessage)
		ruleDiags = ruleDiags.Append(moreDiags)
		refs = append(refs, moreRefs...)

		scope := ctx.EvaluationScope(self, keyData)
		hclCtx, moreDiags := scope.EvalContext(refs)
		ruleDiags = ruleDiags.Append(moreDiags)

		result, hclDiags := rule.Condition.Value(hclCtx)
		ruleDiags = ruleDiags.Append(hclDiags)

		errorValue, errorDiags := rule.ErrorMessage.Value(hclCtx)
		ruleDiags = ruleDiags.Append(errorDiags)

		diags = diags.Append(ruleDiags)

		if ruleDiags.HasErrors() {
			log.Printf("[TRACE] evalCheckRules: %s: %s", typ.FailureSummary(), ruleDiags.Err().Error())
		}

		if !result.IsKnown() {
			continue // We'll wait until we've learned more, then.
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
			continue
		}
		var err error
		result, err = convert.Convert(result, cty.Bool)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    severity,
				Summary:     errInvalidCondition,
				Detail:      fmt.Sprintf("Invalid condition result value: %s.", tfdiags.FormatError(err)),
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		// The condition result may be marked if the expression refers to a
		// sensitive value.
		result, _ = result.Unmark()

		if result.True() {
			continue
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
			Summary:     typ.FailureSummary(),
			Detail:      errorMessage,
			Subject:     rule.Condition.Range().Ptr(),
			Expression:  rule.Condition,
			EvalContext: hclCtx,
		})
	}

	return diags
}
