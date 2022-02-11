package ngaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Reference struct {
	Subject     Referenceable
	SourceRange tfdiags.SourceRange
	Remaining   hcl.Traversal
}

// ParseRef attempts to extract a referencable address from the prefix of the
// given traversal, which must be an absolute traversal or this function
// will panic.
//
// If no error diagnostics are returned, the returned reference includes the
// address that was extracted, the source range it was extracted from, and any
// remaining relative traversal that was not consumed as part of the
// reference.
//
// If error diagnostics are returned then the Reference value is invalid and
// must not be used.
func ParseRef(traversal hcl.Traversal) (*Reference, tfdiags.Diagnostics) {
	ref, diags := parseRef(traversal)

	// Normalize a little to make life easier for callers.
	if ref != nil {
		if len(ref.Remaining) == 0 {
			ref.Remaining = nil
		}
	}

	return ref, diags
}

// ParseRefStr is a helper wrapper around ParseRef that takes a string
// and parses it with the HCL native syntax traversal parser before
// interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a reference string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseRef.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned reference may be nil or incomplete.
func ParseRefStr(str string) (*Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	ref, targetDiags := ParseRef(traversal)
	diags = diags.Append(targetDiags)
	return ref, diags
}

func parseRef(traversal hcl.Traversal) (*Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	root := traversal.RootName()
	rootRange := traversal[0].SourceRange()

	switch root {

	case "each":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     addrs.ForEachAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "local":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     addrs.LocalValue{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "component":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     ComponentCall{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "group":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     ComponentGroupCall{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "var":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     addrs.InputVariable{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			// TODO: didyoumean
			Detail:  fmt.Sprintf("There is no referencable object prefix \"%q.\".", root),
			Subject: &rootRange,
		})
		return nil, diags
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
