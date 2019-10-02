package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/tfdiags"
)

// LocalValue represents a single entry inside a "locals" block in the
// configuration.
type LocalValue struct {
	// Name is the name given for this local value in the configuration.
	//
	// It is guaranteed to be a valid HCL identifier.
	Name string

	// Value is the expression that should produce the local value.
	Value hcl.Expression

	// SrcRange and NameRange are the source locations of the entire
	// declaration and of the name portion of the declaration respectively.
	SrcRange, NameRange tfdiags.SourceRange
}

func decodeLocalValueAttr(attr *hcl.Attribute) (*LocalValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	lv := &LocalValue{
		Name:      attr.Name,
		Value:     attr.Expr,
		SrcRange:  tfdiags.SourceRangeFromHCL(attr.Range),
		NameRange: tfdiags.SourceRangeFromHCL(attr.NameRange),
	}

	if !hclsyntax.ValidIdentifier(lv.Name) {
		// This should never happen in practice because the HCL parser wouldn't
		// permit an invalid name in this position, but we'll be thorough and
		// check it anyway.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid name for local value",
			Detail:   fmt.Sprintf("The name %q is not a valid name for a local value. Must start with a letter, followed by zero or more letters, digits, and underscores.", lv.Name),
			Subject:  attr.NameRange.Ptr(),
		})
	}

	return lv, diags
}
