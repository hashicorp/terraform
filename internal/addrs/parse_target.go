// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ParseAbsTargetable attempts to interpret the given traversal as the concrete
// address of a module instance, resource, or resource instance, which is the
// same syntax as used for a target. The given traversal must be absolute, or
// this function will panic.
//
// A resource instance address without an instance key is returned as an
// AbsResource, which refers to the whole resource.
//
// If error diagnostics are returned then the address is invalid and must not
// be used.
func ParseAbsTargetable(traversal hcl.Traversal) (Targetable, tfdiags.Diagnostics) {
	return parseAbsTarget(traversal, false)
}

// parseAbsTarget parses the concrete address of a module instance, resource,
// or resource instance. If allowPartial is set, any of the instance keys may
// be WildcardKey, for steps written as [*], which indicate a "partial" address
// that refers to all potential instances.
func parseAbsTarget(traversal hcl.Traversal, allowPartial bool) (Targetable, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, allowPartial)
	if diags.HasErrors() {
		return nil, diags
	}

	if len(remain) == 0 {
		return path, diags
	}

	riAddr, moreDiags := parseResourceInstanceUnderModule(path, allowPartial, remain)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	if riAddr.Resource.Key == NoKey {
		// We always assume that a no-key instance is meant to
		// be referring to the whole resource.
		return riAddr.ContainingResource(), diags
	}
	return riAddr, diags
}

// parseConfigResourceUnderModule attempts to parse the given traversal as the
// address for a ConfigResource in the context of the given module.
//
// Error diagnostics are returned if the resource address contains an instance
// key.
func parseConfigResourceUnderModule(moduleAddr Module, remain hcl.Traversal) (ConfigResource, tfdiags.Diagnostics) {
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
		return ConfigResource{}, diags
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
		return ConfigResource{}, diags
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
		return ConfigResource{}, diags
	}

	remain = remain[2:]
	if len(remain) > 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource instance keys not allowed",
			Detail:   "Resource address must be a resource (e.g. \"test_instance.foo\"), not a resource instance (e.g. \"test_instance.foo[1]\").",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return ConfigResource{}, diags
	}
	return ConfigResource{
		Module: moduleAddr,
		Resource: Resource{
			Mode: mode,
			Type: typeName,
			Name: name,
		},
	}, diags
}

func parseResourceInstanceUnderModule(moduleAddr ModuleInstance, allowPartial bool, remain hcl.Traversal) (AbsResourceInstance, tfdiags.Diagnostics) {
	// Note that this helper is used as part of both ParseTarget and
	// ParseMoveEndpoint, so its error messages should be generic
	// enough to suit both situations.

	var diags tfdiags.Diagnostics

	mode := ManagedResourceMode
	switch remain.RootName() {
	case "data":
		mode = DataResourceMode
		remain = remain[1:]
	case "ephemeral":
		mode = EphemeralResourceMode
		remain = remain[1:]
	case "list":
		mode = ListResourceMode
		remain = remain[1:]
	case "resource":
		// Starting a resource address with "resource" is optional, so we'll
		// just ignore it.
		remain = remain[1:]
	case "count", "each", "local", "module", "path", "self", "terraform", "var", "template", "lazy", "arg":
		// These are all reserved words that are not valid as resource types.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   fmt.Sprintf("The keyword %q is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with \"resource.\".", remain.RootName()),
			Subject:  remain.SourceRange().Ptr(),
		})
		return AbsResourceInstance{}, diags
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
		case EphemeralResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "An ephemeral resource type name is required.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
		case ListResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "A list resource type name is required.",
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
		switch tt := remain[0].(type) {
		case hcl.TraverseIndex:
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
		case hcl.TraverseSplat:
			if allowPartial {
				return moduleAddr.ResourceInstance(mode, typeName, name, WildcardKey), diags
			}

			// Otherwise, return an error.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Resource instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			return AbsResourceInstance{}, diags
		default:
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

// ParseAbsTargetableStr is a helper wrapper around ParseAbsTargetable that
// takes a string and parses it with the HCL native syntax traversal parser
// before interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a string is coming from a source that should be identified in error
// messages then the caller should instead parse it directly using a suitable
// function from the HCL API and pass the traversal itself to
// ParseAbsTargetable.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address may be nil or incomplete.
func ParseAbsTargetableStr(str string) (Targetable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	addr, addrDiags := ParseAbsTargetable(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
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
	addr, diags := ParseAbsTargetable(traversal)
	if diags.HasErrors() {
		return AbsResource{}, diags
	}

	switch tt := addr.(type) {

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
	subject, diags := ParseAbsTargetable(traversal)
	if diags.HasErrors() {
		return AbsResourceInstance{}, diags
	}

	addr, validateDiags := validateResourceFromTargetable(subject, traversal.SourceRange().Ptr())
	diags = diags.Append(validateDiags)
	return addr, diags
}

// ParsePartialResourceInstance attempts to interpret the given traversal as a
// partial absolute resource instance address, which may use the [*] wildcard
// syntax for instance keys of the module path and resource.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the AbsResource value is invalid and
// must not be used.
func ParsePartialResourceInstance(traversal hcl.Traversal) (AbsResourceInstance, tfdiags.Diagnostics) {
	subject, diags := parseAbsTarget(traversal, true)
	if diags.HasErrors() {
		return AbsResourceInstance{}, diags
	}

	addr, validateDiags := validateResourceFromTargetable(subject, traversal.SourceRange().Ptr())
	diags = diags.Append(validateDiags)
	return addr, diags
}

func validateResourceFromTargetable(addr Targetable, src *hcl.Range) (AbsResourceInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	switch tt := addr.(type) {

	case AbsResource:
		return tt.Instance(NoKey), diags

	case AbsResourceInstance:
		return tt, diags

	case ModuleInstance: // Catch likely user error with specialized message
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource instance address is required here. The module path must be followed by a resource instance specification.",
			Subject:  src,
		})
		return AbsResourceInstance{}, diags

	default: // Generic message for other address types
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "A resource address is required here.",
			Subject:  src,
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

// ParsePartialResourceInstanceStr is a helper wrapper around
// ParsePartialResourceInstance that takes a string and parses it with the HCL
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
func ParsePartialResourceInstanceStr(str string) (AbsResourceInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalPartial([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsResourceInstance{}, diags
	}

	addr, addrDiags := ParsePartialResourceInstance(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}
