package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Reference describes a reference expression found in the configuration,
// capturing what it referred to and where it was found in source code.
type Reference struct {
	Target      Referenceable
	SourceRange tfdiags.SourceRange
}

// ParseReference raises a raw absolute traversal into a higher-level reference,
// or returns error diagnostics explaining why it cannot.
//
// The returned traversal is a relative traversal covering the remainder of
// the given traversal after the part captured into the returned reference,
// in case the caller wants to do further validation or analysis of the
// subsequent steps.
func ParseReference(traversal hcl.Traversal) (Reference, hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret Reference
	switch rootName := traversal.RootName(); rootName {

	case "var":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		ret.Target = InputVariable{Name: name}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)
		return ret, remain, diags

	case "local":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		ret.Target = LocalValue{Name: name}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)
		return ret, remain, diags

	case "component":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		ret.Target = Component{Name: name}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)
		return ret, remain, diags

	case "stack":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		ret.Target = StackCall{Name: name}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)
		return ret, remain, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unknown symbol",
			Detail:   fmt.Sprintf("There is no symbol %q defined in the current scope.", rootName),
			Subject:  traversal[0].SourceRange().Ptr(),
		})
		return ret, nil, diags
	}
}

func parseSingleAttrRef(traversal hcl.Traversal) (string, hcl.Range, hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	root := traversal.RootName()
	rootRange := traversal[0].SourceRange()

	if len(traversal) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("The %q object cannot be accessed directly. Instead, access one of its attributes.", root),
			Subject:  &rootRange,
		})
		return "", hcl.Range{}, nil, diags
	}
	if attrTrav, ok := traversal[1].(hcl.TraverseAttr); ok {
		return attrTrav.Name, hcl.RangeBetween(rootRange, attrTrav.SrcRange), traversal[2:], diags
	}
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid reference",
		Detail:   fmt.Sprintf("The %q object does not support this operation.", root),
		Subject:  traversal[1].SourceRange().Ptr(),
	})
	return "", hcl.Range{}, nil, diags
}
