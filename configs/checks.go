package configs

import (
	"fmt"
	"unicode"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// CheckRule represents a configuration-defined validation rule, precondition,
// or postcondition. Blocks of this sort can appear in a few different places
// in configuration, including "validation" blocks for variables,
// and "precondition" and "postcondition" blocks for resources.
type CheckRule struct {
	// Condition is an expression that must evaluate to true if the condition
	// holds or false if it does not. If the expression produces an error then
	// that's considered to be a bug in the module defining the check.
	//
	// The available variables in a condition expression vary depending on what
	// a check is attached to. For example, validation rules attached to
	// input variables can only refer to the variable that is being validated.
	Condition hcl.Expression

	// ErrorMessage is one or more full sentences, which would need to be in
	// English for consistency with the rest of the error message output but
	// can in practice be in any language as long as it ends with a period.
	// The message should describe what is required for the condition to return
	// true in a way that would make sense to a caller of the module.
	ErrorMessage string

	DeclRange hcl.Range
}

// decodeCheckRuleBlock decodes the contents of the given block as a check rule.
//
// Unlike most of our "decode..." functions, this one can be applied to blocks
// of various types as long as their body structures are "check-shaped". The
// function takes the containing block only because some error messages will
// refer to its location, and the returned object's DeclRange will be the
// block's header.
func decodeCheckRuleBlock(block *hcl.Block, override bool) (*CheckRule, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cr := &CheckRule{
		DeclRange: block.DefRange,
	}

	if override {
		// For now we'll just forbid overriding check blocks, to simplify
		// the initial design. If we can find a clear use-case for overriding
		// checks in override files and there's a way to define it that
		// isn't confusing then we could relax this.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Can't override %s blocks", block.Type),
			Detail:   fmt.Sprintf("Override files cannot override %q blocks.", block.Type),
			Subject:  cr.DeclRange.Ptr(),
		})
		return cr, diags
	}

	content, moreDiags := block.Body.Content(checkRuleBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["condition"]; exists {
		cr.Condition = attr.Expr
	}

	if attr, exists := content.Attributes["error_message"]; exists {
		moreDiags := gohcl.DecodeExpression(attr.Expr, nil, &cr.ErrorMessage)
		diags = append(diags, moreDiags...)
		if !moreDiags.HasErrors() {
			const errSummary = "Invalid validation error message"
			switch {
			case cr.ErrorMessage == "":
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail:   "An empty string is not a valid nor useful error message.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			case !looksLikeSentences(cr.ErrorMessage):
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
					Summary:  errSummary,
					Detail:   "Validation error message must be at least one full English sentence starting with an uppercase letter and ending with a period or question mark.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	return cr, diags
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

var checkRuleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "condition",
			Required: true,
		},
		{
			Name:     "error_message",
			Required: true,
		},
	},
}
