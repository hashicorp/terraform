package lang

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// EvalCheckErrorMessage makes a best effort to evaluate the given expression,
// as an error message string as we'd expect for an error_message argument
// inside a validation/condition/check block.
//
// It will either return a non-empty message string or it'll return diagnostics
// with either errors or warnings that explain why the given expression isn't
// acceptable.
func EvalCheckErrorMessage(expr hcl.Expression, hclCtx *hcl.EvalContext) (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	val, hclDiags := expr.Value(hclCtx)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return "", diags
	}

	val, err := convert.Convert(val, cty.String)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid error message",
			Detail:      fmt.Sprintf("Unsuitable value for error message: %s.", tfdiags.FormatError(err)),
			Subject:     expr.Range().Ptr(),
			Expression:  expr,
			EvalContext: hclCtx,
		})
		return "", diags
	}
	if !val.IsKnown() {
		return "", diags
	}
	if val.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid error message",
			Detail:      "Unsuitable value for error message: must not be null.",
			Subject:     expr.Range().Ptr(),
			Expression:  expr,
			EvalContext: hclCtx,
		})
		return "", diags
	}

	val, valMarks := val.Unmark()
	if _, sensitive := valMarks[marks.Sensitive]; sensitive {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Error message refers to sensitive values",
			Detail: `The error expression used to explain this condition refers to sensitive values, so Terraform will not display the resulting message.

You can correct this by removing references to sensitive values, or by carefully using the nonsensitive() function if the expression will not reveal the sensitive data.`,
			Subject:     expr.Range().Ptr(),
			Expression:  expr,
			EvalContext: hclCtx,
		})
		return "", diags
	}

	// NOTE: We've discarded any other marks the string might have been carrying,
	// aside from the sensitive mark.

	return strings.TrimSpace(val.AsString()), diags
}
