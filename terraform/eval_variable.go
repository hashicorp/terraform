package terraform

import (
	"fmt"
	"log"
	"unicode"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// evalVariableValidations ensures that all of the configured custom validations
// for a variable are passing.
//
// This must be used only after any side-effects that make the value of the
// variable available for use in expression evaluation, such as
// EvalModuleCallArgument for variables in descendent modules.
func evalVariableValidations(addr addrs.AbsInputVariableInstance, config *configs.Variable, expr hcl.Expression, ctx EvalContext) (diags tfdiags.Diagnostics) {
	if config == nil || len(config.Validations) == 0 {
		log.Printf("[TRACE] evalVariableValidations: not active for %s, so skipping", addr)
		return nil
	}

	// Variable nodes evaluate in the parent module to where they were declared
	// because the value expression (expr, if set) comes from the calling
	// "module" block in the parent module.
	//
	// Validation expressions are statically validated (during configuration
	// loading) to refer only to the variable being validated, so we can
	// bypass our usual evaluation machinery here and just produce a minimal
	// evaluation context containing just the required value, and thus avoid
	// the problem that ctx's evaluation functions refer to the wrong module.
	val := ctx.GetVariableValue(addr)
	hclCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(map[string]cty.Value{
				config.Name: val,
			}),
		},
		Functions: ctx.EvaluationScope(nil, EvalDataForNoInstanceKey).Functions(),
	}

	for _, validation := range config.Validations {
		const errInvalidCondition = "Invalid variable validation result"
		const errInvalidValue = "Invalid value for variable"
		const errInvalidErrorMessage = "Invalid validation error message"

		result, moreDiags := validation.Condition.Value(hclCtx)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition expression failed: %s", addr, validation.DeclRange, diags.Err().Error())
		}
		if !result.IsKnown() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition value is unknown, so skipping validation for now", addr, validation.DeclRange)
			continue // We'll wait until we've learned more, then.
		}
		if result.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      "Validation condition expression must return either true or false, not null.",
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}
		var err error
		result, err = convert.Convert(result, cty.Bool)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      fmt.Sprintf("Invalid validation condition result value: %s.", tfdiags.FormatError(err)),
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		errorMessage, moreDiags := validation.ErrorMessage.Value(hclCtx)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition expression failed: %s", addr, validation.DeclRange, diags.Err().Error())
		}
		if !errorMessage.IsKnown() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition value is unknown, so skipping validation for now", addr, validation.DeclRange)
			continue // We'll wait until we've learned more, then.
		}
		if errorMessage.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      "Validation error message expression must return a string, not null.",
				Subject:     validation.ErrorMessage.Range().Ptr(),
				Expression:  validation.ErrorMessage,
				EvalContext: hclCtx,
			})
			continue
		}

		errorMessage, err = convert.Convert(errorMessage, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidErrorMessage,
				Detail:      fmt.Sprintf("Invalid validation error message result value: %s.", tfdiags.FormatError(err)),
				Subject:     validation.ErrorMessage.Range().Ptr(),
				Expression:  validation.ErrorMessage,
				EvalContext: hclCtx,
			})
			continue
		}

		// Validation condition may be marked if the input variable is bound to
		// a sensitive value. This is irrelevant to the validation process, so
		// we discard the marks now.
		result, _ = result.Unmark()

		if result.False() {
			if expr != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidValue,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage.AsString(), validation.DeclRange.String()),
					Subject:  expr.Range().Ptr(),
				})
			} else {
				// Since we don't have a source expression for a root module
				// variable, we'll just report the error from the perspective
				// of the variable declaration itself.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidValue,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage.AsString(), validation.DeclRange.String()),
					Subject:  config.DeclRange.Ptr(),
				})
			}
		}

		if errorMessage.Type() != cty.String {
			if expr != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidErrorMessage,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage.AsString(), validation.DeclRange.String()),
					Subject:  expr.Range().Ptr(),
				})
			} else {
				// Since we don't have a source expression for a root module
				// variable, we'll just report the error from the perspective
				// of the variable declaration itself.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidErrorMessage,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage.AsString(), validation.DeclRange.String()),
					Subject:  config.DeclRange.Ptr(),
				})
			}
		}

		switch {
		case errorMessage.AsString() == "":
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errInvalidErrorMessage,
				Detail:   "An empty string is not a valid nor useful error message.",
				Subject:  config.DeclRange.Ptr(),
			})
		case !looksLikeSentences(errorMessage.AsString()):
			// Because we're going to include this string verbatim as part
			// of a bigger error message written in our usual style in
			// English, we'll require the given error message to conform
			// to that. We might relax this in future if e.g. we start
			// presenting these error messages in a different way, or if
			// Terraform starts supporting producing error messages in
			// other human languages, etc.
			// For pragmatism we also allow sentences ending with
			// exclamation points, but we don't mention it explicitly here
			// because that's not really consistent with the Terraform UI
			// writing style.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errInvalidErrorMessage,
				Detail:   "Validation error message must be at least one full English sentence starting with an uppercase letter and ending with a period or question mark.",
				Subject:  config.DeclRange.Ptr(),
			})
		}
	}

	return diags
}

// looksLikeSentence is a simple heuristic that encourages writing error
// messages that will be presentable when included as part of a larger
// Terraform error diagnostic whose other text is written in the Terraform
// UI writing style.
//
// This is intentionally not a very strong validation since we're assuming
// that module authors want to write good messages and might just need a nudge
// about Terraform's specific style, rather than that they are going to try
// to work around these rules to write a lower-quality message.
func looksLikeSentences(s string) bool {
	if len(s) < 1 {
		return false
	}
	runes := []rune(s) // HCL guarantees that all strings are valid UTF-8
	first := runes[0]
	last := runes[len(runes)-1]

	// If the first rune is a letter then it must be an uppercase letter.
	// (This will only see the first rune in a multi-rune combining sequence,
	// but the first rune is generally the letter if any are, and if not then
	// we'll just ignore it because we're primarily expecting English messages
	// right now anyway, for consistency with all of Terraform's other output.)
	if unicode.IsLetter(first) && !unicode.IsUpper(first) {
		return false
	}

	// The string must be at least one full sentence, which implies having
	// sentence-ending punctuation.
	// (This assumes that if a sentence ends with quotes then the period
	// will be outside the quotes, which is consistent with Terraform's UI
	// writing style.)
	return last == '.' || last == '?' || last == '!'
}
