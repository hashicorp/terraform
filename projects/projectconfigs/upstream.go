package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/tfdiags"
)

// Upstream represents a remote workspace from another project that this
// project derives values from.
type Upstream struct {
	// Name is the name label given in the block header.
	//
	// It is guaranteed to be a valid HCL identifier.
	Name string

	// ForEach is the expression given in the for_each argument, or nil if
	// that argument wasn't set.
	ForEach hcl.Expression

	// Remote is the expression given in the "remote" argument.
	Remote hcl.Expression

	// DeclRange is the source range of the block header of this block,
	// for use in diagnostic messages. NameRange is the range of the
	// Name string specifically.
	DeclRange, NameRange tfdiags.SourceRange
}

func decodeUpstreamBlock(block *hcl.Block) (*Upstream, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	u := &Upstream{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
		NameRange: tfdiags.SourceRangeFromHCL(block.LabelRanges[0]),
	}

	if !hclsyntax.ValidIdentifier(u.Name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid name for \"upstream\" block",
			Detail:   fmt.Sprintf("The name %q is not a valid name for an \"upstream\" block. Must start with a letter, followed by zero or more letters, digits, and underscores.", u.Name),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	content, hclDiags := block.Body.Content(workspaceSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["for_each"]; ok {
		u.ForEach = attr.Expr
	}

	if attr, ok := content.Attributes["remote"]; ok {
		u.Remote = attr.Expr
	}

	return u, diags
}

var upstreamSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "for_each"},
		{Name: "remote", Required: true},
	},
}
