// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ParseAbsTargetable attempts to interpret the given traversal as the concrete
// address of a module instance, resource, or resource instance, which is the
// same syntax as used for a target. The given traversal must be absolute, or
// this function will panic.
//
// Unlike a target, which is parsed as a TargetPattern, a module call without an
// instance key only refers to the instance of that call without count or
// for_each, and a resource instance address without an instance key is
// returned as an AbsResource, which refers to the whole resource.
//
// If error diagnostics are returned then the address is invalid and must not
// be used.
func ParseAbsTargetable(traversal hcl.Traversal) (Targetable, tfdiags.Diagnostics) {
	return parseAbsTarget(traversal, knownInstanceKeys)
}

// ParseTarget attempts to interpret the given traversal as a target address.
// The given traversal must be absolute, or this function will panic.
//
// The traversal may be a traversal pattern, such as those produced by
// hclsyntax.ParseTraversalPartial, in which case a [*] step selects every
// instance.
//
// A target is always a TargetPattern, so any module call or resource written
// without an instance key selects every instance, the same as using an explicit
// [*] wildcard. Use ParseTargetAction for actions, and ParseAbsTargetable to
// parse the concrete address of a single module instance, resource, or resource
// instance.
//
// If error diagnostics are returned then the TargetPattern is invalid and must
// not be used.
func ParseTarget(traversal hcl.Traversal) (TargetPattern, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, wildcardInstanceKeys)
	if diags.HasErrors() {
		return TargetPattern{}, diags
	}

	if len(remain) == 0 {
		return newTargetPattern(path, moduleTargetShape, Resource{}, Action{}, NoKey), diags
	}

	riAddr, moreDiags := parseResourceInstanceUnderModule(path, wildcardInstanceKeys, remain)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return TargetPattern{}, diags
	}

	return newTargetPattern(riAddr.Module, resourceTargetShape, riAddr.Resource.Resource, Action{}, riAddr.Resource.Key), diags
}

// parseAbsTarget parses the concrete address of a module instance, resource,
// or resource instance, with instance keys as allowed by keys. If wildcards
// are allowed, any of the instance keys may be WildcardKey, for steps written
// as [*], which indicate a "partial" address that refers to all potential
// instances.
func parseAbsTarget(traversal hcl.Traversal, keys instanceKeys) (Targetable, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, keys)
	if diags.HasErrors() {
		return nil, diags
	}

	if len(remain) == 0 {
		return path, diags
	}

	riAddr, moreDiags := parseResourceInstanceUnderModule(path, keys, remain)
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
	resource, _, _, diags := parseResourceUnderModule(remain, noInstanceKeys, false)
	if diags.HasErrors() {
		return ConfigResource{}, diags
	}
	return resource.InModule(moduleAddr), diags
}

// parseResourceInstanceUnderModule attempts to parse the given traversal as
// the address of a resource instance within the given module instance, with
// instance keys as allowed by keys. An address without an instance key is
// parsed as the instance with NoKey.
func parseResourceInstanceUnderModule(moduleAddr ModuleInstance, keys instanceKeys, remain hcl.Traversal) (AbsResourceInstance, tfdiags.Diagnostics) {
	resource, key, _, diags := parseResourceUnderModule(remain, keys, false)
	if diags.HasErrors() {
		return AbsResourceInstance{}, diags
	}
	return resource.Instance(key).Absolute(moduleAddr), diags
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

// ParseTargetStr is a helper wrapper around ParseTarget that takes a string and
// parses it with the HCL native syntax traversal pattern parser before
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
// the returned target may be incomplete.
func ParseTargetStr(str string) (TargetPattern, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalPartial([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return TargetPattern{}, diags
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
	subject, diags := parseAbsTarget(traversal, wildcardInstanceKeys)
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
