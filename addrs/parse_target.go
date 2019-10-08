package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/tfdiags"
)

// Target describes a targeted address with source location information.
type Target struct {
	Subject     Targetable
	SourceRange tfdiags.SourceRange
}

// ParseTarget attempts to interpret the given traversal as a targetable
// address. The given traversal must be absolute, or this function will
// panic.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the Target value is invalid and
// must not be used.
func ParseTarget(traversal hcl.Traversal) (*Target, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal)
	if diags.HasErrors() {
		return nil, diags
	}

	rng := tfdiags.SourceRangeFromHCL(traversal.SourceRange())

	if len(remain) == 0 {
		return &Target{
			Subject:     path,
			SourceRange: rng,
		}, diags
	}

	mode := ManagedResourceMode
	if remain.RootName() == "data" {
		mode = DataResourceMode
		remain = remain[1:]
	}

	if len(remain) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Resource specification must include a resource type and name.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return nil, diags
	}

	var typeName, name string
	switch tt := remain[0].(type) {
	case hcl.TraverseRoot:
		typeName = tt.Name
	case hcl.TraverseAttr:
		typeName = tt.Name
	default:
		switch mode {
		case ManagedResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "A resource type name is required.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
		case DataResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "A data source name is required.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
		default:
			panic("unknown mode")
		}
		return nil, diags
	}

	switch tt := remain[1].(type) {
	case hcl.TraverseAttr:
		name = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource name is required.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return nil, diags
	}

	var subject Targetable
	remain = remain[2:]
	switch len(remain) {
	case 0:
		subject = path.Resource(mode, typeName, name)
	case 1:
		if tt, ok := remain[0].(hcl.TraverseIndex); ok {
			key, err := ParseInstanceKey(tt.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid address",
					Detail:   fmt.Sprintf("Invalid resource instance key: %s.", err),
					Subject:  remain[0].SourceRange().Ptr(),
				})
				return nil, diags
			}

			subject = path.ResourceInstance(mode, typeName, name, key)
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Resource instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			return nil, diags
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return nil, diags
	}

	return &Target{
		Subject:     subject,
		SourceRange: rng,
	}, diags
}

// ParseTargetStr is a helper wrapper around ParseTarget that takes a string
// and parses it with the HCL native syntax traversal parser before
// interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a target string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseTarget.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned target may be nil or incomplete.
func ParseTargetStr(str string) (*Target, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	target, targetDiags := ParseTarget(traversal)
	diags = diags.Append(targetDiags)
	return target, diags
}

// ParseAbsResource attempts to interpret the given traversal as an absolute
// resource address, using the same syntax as expected by ParseTarget.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the AbsResource value is invalid and
// must not be used.
func ParseAbsResource(traversal hcl.Traversal) (AbsResource, tfdiags.Diagnostics) {
	addr, diags := ParseTarget(traversal)
	if diags.HasErrors() {
		return AbsResource{}, diags
	}

	switch tt := addr.Subject.(type) {

	case AbsResource:
		return tt, diags

	case AbsResourceInstance: // Catch likely user error with specialized message
		// Assume that the last element of the traversal must be the index,
		// since that's required for a valid resource instance address.
		indexStep := traversal[len(traversal)-1]
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource address is required. This instance key identifies a specific resource instance, which is not expected here.",
			Subject:  indexStep.SourceRange().Ptr(),
		})
		return AbsResource{}, diags

	case ModuleInstance: // Catch likely user error with specialized message
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource address is required here. The module path must be followed by a resource specification.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsResource{}, diags

	default: // Generic message for other address types
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource address is required here.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsResource{}, diags

	}
}

// ParseAbsResourceStr is a helper wrapper around ParseAbsResource that takes a
// string and parses it with the HCL native syntax traversal parser before
// interpreting it.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address may be incomplete.
//
// Since this function has no context about the source of the given string,
// any returned diagnostics will not have meaningful source location
// information.
func ParseAbsResourceStr(str string) (AbsResource, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsResource{}, diags
	}

	addr, addrDiags := ParseAbsResource(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

// ParseAbsResourceInstance attempts to interpret the given traversal as an
// absolute resource instance address, using the same syntax as expected by
// ParseTarget.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the AbsResource value is invalid and
// must not be used.
func ParseAbsResourceInstance(traversal hcl.Traversal) (AbsResourceInstance, tfdiags.Diagnostics) {
	addr, diags := ParseTarget(traversal)
	if diags.HasErrors() {
		return AbsResourceInstance{}, diags
	}

	switch tt := addr.Subject.(type) {

	case AbsResource:
		return tt.Instance(NoKey), diags

	case AbsResourceInstance:
		return tt, diags

	case ModuleInstance: // Catch likely user error with specialized message
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource instance address is required here. The module path must be followed by a resource instance specification.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsResourceInstance{}, diags

	default: // Generic message for other address types
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource address is required here.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsResourceInstance{}, diags

	}
}

// ParseAbsResourceInstanceStr is a helper wrapper around
// ParseAbsResourceInstance that takes a string and parses it with the HCL
// native syntax traversal parser before interpreting it.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address may be incomplete.
//
// Since this function has no context about the source of the given string,
// any returned diagnostics will not have meaningful source location
// information.
func ParseAbsResourceInstanceStr(str string) (AbsResourceInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsResourceInstance{}, diags
	}

	addr, addrDiags := ParseAbsResourceInstance(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}
