package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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

	// Description is a human-oriented description of the purpose of this
	// context value, written in full sentences in a natural language.
	Description string

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

	content, hclDiags := block.Body.Content(contextSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["type"]; ok {
		ty, hclDiags := typeexpr.TypeConstraint(attr.Expr)
		diags = diags.Append(hclDiags)
		cv.Type = ty
	} else {
		cv.Type = cty.DynamicPseudoType
	}

	if attr, ok := content.Attributes["default"]; ok {
		cv.Default = attr.Expr
		if traversals := attr.Expr.Variables(); len(traversals) > 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid context default value",
				Detail:   "A default value must be a constant, and so cannot refer to any other objects.",
				Subject:  traversals[0].SourceRange().Ptr(),
				Context:  attr.Range.Ptr(),
			})
		}
	}

	if attr, ok := content.Attributes["description"]; ok {
		// We don't allow variables/functions in the description because
		// we want to be able to display it in a UI that might be prompting
		// for data that would be needed to evaluate expressions, so we'll
		// just evaluate this right now and HCL will generate "variables cannot
		// be used here" errors in case a user tries.
		val, hclDiags := attr.Expr.Value(nil)
		diags = diags.Append(hclDiags)
		if !diags.HasErrors() {
			if val.Type().Equals(cty.String) && !val.IsNull() {
				cv.Description = val.AsString()
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid description for context value",
					Detail:   "The description of a context value must be a string and should contain a natural language description of the meaning of this context value using full sentences.",
					Subject:  attr.Expr.StartRange().Ptr(),
				})
			}
		}
	}

	return cv, diags
}

var contextSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "type"},
		{Name: "default"},
		{Name: "description"},
	},
}
