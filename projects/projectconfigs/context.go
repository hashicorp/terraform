package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/tfdiags"
)

// ContextValue represents a "context" block in the configuration, describing
// a value that can (or possibly must) be set to allow the workspace
// configuration to tailor itself to a specific execution context.
//
// For example, a context value might select the location of a service that
// the workspace configurations depend on but that might be accessed
// differently during development than in production.
//
// Best to keep use of this feature to a minimum, but it's here for pragmatic
// reasons knowing that sometimes it's not possible to make a fully-portable
// Terraform project configuration.
type ContextValue struct {
	// Name is the name given for this context value in the configuration.
	//
	// It is guaranteed to be a valid HCL identifier.
	Name string

	// Type is the type constraint that given values must conform to.
	Type cty.Type

	// Default is a default value for this context value, or nil if
	// this is a required context value.
	Default hcl.Expression

	// DeclRange is the source range of the block header of this block,
	// for use in diagnostic messages. NameRange is the range of the
	// Name string specifically.
	DeclRange, NameRange tfdiags.SourceRange
}

func decodeContextBlock(block *hcl.Block) (*ContextValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	cv := &ContextValue{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
		NameRange: tfdiags.SourceRangeFromHCL(block.LabelRanges[0]),
	}

	if !hclsyntax.ValidIdentifier(cv.Name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid name for context value",
			Detail:   fmt.Sprintf("The name %q is not a valid name for a context value. Must start with a letter, followed by zero or more letters, digits, and underscores.", cv.Name),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	return cv, diags
}
