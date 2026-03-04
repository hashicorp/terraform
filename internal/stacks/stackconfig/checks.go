// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import "github.com/hashicorp/hcl/v2"

// CheckRule represents a custom validation rule for a stack input variable.
//
// This is the stacks-specific equivalent of configs.CheckRule in the core
// Terraform package. It is intentionally duplicated here to maintain
// separation between stacks and core Terraform, allowing each to evolve
// independently.
type CheckRule struct {
	// Condition is an expression that must evaluate to true if the validation
	// passes, or false if it fails. The expression may only refer to the
	// variable being validated (via var.<name>).
	Condition hcl.Expression

	// ErrorMessage is an expression that evaluates to the error message shown
	// to the user when the condition is false. It must evaluate to a string.
	ErrorMessage hcl.Expression

	DeclRange hcl.Range
}

// decodeCheckRuleBlock decodes a validation block for stack input variables.
// This is duplicated from the core configs package to maintain separation between
// stacks and core Terraform, allowing each to evolve independently.
func decodeCheckRuleBlock(block *hcl.Block) (*CheckRule, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cr := &CheckRule{
		DeclRange: block.DefRange,
	}

	content, hclDiags := block.Body.Content(checkRuleBlockSchema)
	diags = append(diags, hclDiags...)

	if attr, exists := content.Attributes["condition"]; exists {
		cr.Condition = attr.Expr

		if len(cr.Condition.Variables()) == 0 {
			// A condition expression that doesn't refer to any variable is
			// pointless, because its result would always be a constant.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid validation expression",
				Detail:   "The condition expression must refer to at least one object from elsewhere in the configuration, or else its result would not be checking anything.",
				Subject:  cr.Condition.Range().Ptr(),
			})
		}
	}

	if attr, exists := content.Attributes["error_message"]; exists {
		cr.ErrorMessage = attr.Expr
	}

	return cr, diags
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
