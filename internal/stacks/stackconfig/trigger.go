// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Trigger represents a declaration of a VCS trigger. It can be used to customise the behaviour of
// when a (speculative) plan should be run, similar to orchestrate blocks
type Trigger struct {
	Name string

	// No for-each for now
	// ForEach hcl.Expression

	// Check is an expression that should produce a boolean indicating if a plan should be run.
	// It has a special context to access e.g. branch name / PR target / etc.
	CheckAsStringExpr string

	// IsSpeculativePlanExpr is an expression that should produce a boolean indicating if a plan should be speculative or not.
	// It has a special context to access e.g. branch name / PR target / etc.
	IsSpeculativePlan hcl.Expression

	DeclRange tfdiags.SourceRange
}

func decodeTriggerBlock(file *hcl.File, block *hcl.Block) (*Trigger, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Trigger{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	// TODO: Do we need a name? It is probably not addressable, right?
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid trigger name",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(triggerBlockSchema)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	if attr, ok := content.Attributes["check"]; ok {
		ret.CheckAsStringExpr = string(attr.Expr.Range().SliceBytes(file.Bytes))
	}
	if attr, ok := content.Attributes["is_speculative"]; ok {
		ret.IsSpeculativePlan = attr.Expr
	}

	return ret, diags
}

var triggerBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		// {Name: "for_each", Required: false},
		{Name: "check", Required: false},
		{Name: "is_speculative", Required: false},
	},
}
