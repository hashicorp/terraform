package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	riAddr, moreDiags := parseResourceInstanceUnderModule(path, remain)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	var subject Targetable
	switch {
	case riAddr.Resource.Key == NoKey:
		// We always assume that a no-key instance is meant to
		// be referring to the whole resource, because the distinction
		// doesn't really matter for targets anyway.
		subject = riAddr.ContainingResource()
	default:
		subject = riAddr
	}

	return &Target{
		Subject:     subject,
		SourceRange: rng,
	}, diags
}

func parseResourceInstanceUnderModule(moduleAddr ModuleInstance, remain hcl.Traversal) (AbsResourceInstance, tfdiags.Diagnostics) {
	// Note that this helper is used as part of both ParseTarget and
	// ParseMoveEndpoint, so its error messages should be generic
	// enough to suit both situations.

	var diags tfdiags.Diagnostics

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
		return AbsResourceInstance{}, diags
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
		return AbsResourceInstance{}, diags
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
		return AbsResourceInstance{}, diags
	}

	remain = remain[2:]
	switch len(remain) {
	case 0:
		return moduleAddr.ResourceInstance(mode, typeName, name, NoKey), diags
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
				return AbsResourceInstance{}, diags
			}

			return moduleAddr.ResourceInstance(mode, typeName, name, key), diags
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Resource instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			return AbsResourceInstance{}, diags
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsResourceInstance{}, diags
	}
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

// ModuleAddr returns the module address portion of the subject of
// the recieving target.
//
// Regardless of specific address type, all targets always include
// a module address. They might also include something in that
// module, which this method always discards if so.
func (t *Target) ModuleAddr() ModuleInstance {
	switch addr := t.Subject.(type) {
	case ModuleInstance:
		return addr
	case Module:
		// We assume that a module address is really
		// referring to a module path containing only
		// single-instance modules.
		return addr.UnkeyedInstanceShim()
	case AbsResourceInstance:
		return addr.Module
	case AbsResource:
		return addr.Module
	default:
		// The above cases should be exhaustive for all
		// implementations of Targetable.
		panic(fmt.Sprintf("unsupported target address type %T", addr))
	}
}
