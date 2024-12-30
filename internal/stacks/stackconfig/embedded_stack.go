// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EmbeddedStack describes a call to another stack configuration whose
// declarations should be included as part of the overall stack configuration
// tree.
//
// An embedded stack exists only as a child of another stack and doesn't have
// its own independent identity outside of that calling stack.
//
// HCP Terraform offers a related concept of "linked stacks" where the
// deployment configuration for one stack can refer to the outputs of another,
// while the other stack retains its own independent identity and lifecycle,
// but that concept only makes sense in an environment like HCP Terraform
// where the stack outputs can be published for external consumption.
type EmbeddedStack struct {
	Name string

	SourceAddr                               sourceaddrs.Source
	VersionConstraints                       constraints.IntersectionSpec
	SourceAddrRange, VersionConstraintsRange tfdiags.SourceRange

	ForEach hcl.Expression

	// Inputs is an expression that should produce a value that can convert
	// to an object type derived from the child stack's input variable
	// declarations, and whose attribute values will then be used to populate
	// those input variables.
	Inputs hcl.Expression

	DependsOn []hcl.Traversal

	DeclRange tfdiags.SourceRange
}

func decodeEmbeddedStackBlock(block *hcl.Block) (*EmbeddedStack, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &EmbeddedStack{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid name for call to embedded stack",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(embeddedStackBlockSchema)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	sourceAddr, versionConstraints, moreDiags := decodeSourceAddrArguments(
		content.Attributes["source"],
		content.Attributes["version"],
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	ret.SourceAddr = sourceAddr
	ret.VersionConstraints = versionConstraints
	ret.SourceAddrRange = tfdiags.SourceRangeFromHCL(content.Attributes["source"].Range)
	if content.Attributes["version"] != nil {
		ret.VersionConstraintsRange = tfdiags.SourceRangeFromHCL(content.Attributes["version"].Range)
	}
	// Now that we've populated the mandatory source location fields we can
	// safely return a partial ret if we encounter any further errors, as
	// long as we leave the other fields either unset or in some other
	// reasonable state for careful partial analysis.

	if attr, ok := content.Attributes["for_each"]; ok {
		ret.ForEach = attr.Expr
	}
	if attr, ok := content.Attributes["inputs"]; ok {
		ret.Inputs = attr.Expr
	}
	if attr, ok := content.Attributes["depends_on"]; ok {
		ret.DependsOn, hclDiags = configs.DecodeDependsOn(attr)
		diags = diags.Append(hclDiags)
	}

	return ret, diags
}

var embeddedStackBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version", Required: false},
		{Name: "for_each", Required: false},
		{Name: "inputs", Required: false},
		{Name: "depends_on", Required: false},
	},
}
