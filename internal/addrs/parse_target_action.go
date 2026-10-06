// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// parseAbsActionTarget interprets the given traversal as the concrete address
// of an action or action instance. The given traversal must be absolute, or
// this function will panic.
//
// An action instance address without an instance key is returned as an
// AbsAction, which refers to the whole action.
//
// If error diagnostics are returned then the address is invalid and must not
// be used.
//
// This function matches the behaviour of ParseAbsTargetable, except we are
// ensuring the caller is explicit about what kind of address they want to get.
// We prevent callers accidentally including actions where they shouldn't be
// accessible by keeping these methods separate.
func parseAbsActionTarget(traversal hcl.Traversal) (Targetable, tfdiags.Diagnostics) {
	addr, diags := parseTargetAction(traversal, knownInstanceKeys)
	if diags.HasErrors() {
		return nil, diags
	}

	if addr.Action.Key == NoKey {
		// An action without an instance key refers to the whole action.
		return addr.ContainingAction(), diags
	}
	return addr, diags
}

// ParseTargetAction attempts to interpret the given traversal as an action
// target address. The given traversal must be absolute, or this function will
// panic.
//
// The traversal may be a traversal pattern, such as those produced by
// hclsyntax.ParseTraversalPartial, in which case a [*] step selects every
// instance.
//
// As with other targets, an action target is always parsed as a TargetPattern,
// so any module call or action written without an instance key selects every
// instance, the same as using an explicit [*] wildcard.
//
// If error diagnostics are returned then the TargetPattern is invalid and must
// not be used.
//
// This function matches the behaviour of ParseTarget, except we are ensuring
// the caller is explicit about what kind of target they want to get. We prevent
// callers accidentally including action targets where they shouldn't be
// accessible by keeping these methods separate.
func ParseTargetAction(traversal hcl.Traversal) (TargetPattern, tfdiags.Diagnostics) {
	addr, diags := parseTargetAction(traversal, wildcardInstanceKeys)
	if diags.HasErrors() {
		return TargetPattern{}, diags
	}

	return newTargetPattern(addr.Module, actionTargetShape, Resource{}, addr.Action.Action, addr.Action.Key), diags
}

// parseTargetAction parses an action instance address, with instance keys as
// allowed by keys.
func parseTargetAction(traversal hcl.Traversal, keys instanceKeys) (AbsActionInstance, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, keys)
	if diags.HasErrors() {
		return AbsActionInstance{}, diags
	}

	if len(remain) == 0 {
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action addresses must contain an action reference after the module reference.",
			Subject:  traversal.SourceRange().Ptr(),
		})
	}

	addr, moreDiags := parseActionInstanceUnderModule(path, remain, keys)
	return addr, diags.Append(moreDiags)
}

// ParseTargetActionStr is a helper wrapper around ParseTargetAction that takes
// a string and parses it with the HCL native syntax traversal pattern parser
// before interpreting it.
//
// All the same cautions apply to this as with the equivalent ParseTargetStr.
func ParseTargetActionStr(str string) (TargetPattern, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalPartial([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return TargetPattern{}, diags
	}

	target, targetDiags := ParseTargetAction(traversal)
	diags = diags.Append(targetDiags)
	return target, diags
}

func parseActionInstanceUnderModule(moduleAddr ModuleInstance, remain hcl.Traversal, keys instanceKeys) (AbsActionInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if remain.RootName() != "action" {
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action specification must start with `action`.",
			Subject:  remain.SourceRange().Ptr(),
		})
	}

	remain = remain[1:]

	if len(remain) < 2 {
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action specification must include an action type and name.",
			Subject:  remain.SourceRange().Ptr(),
		})
	}

	var typeName, name string
	switch tt := remain[0].(type) {
	case hcl.TraverseRoot:
		typeName = tt.Name
	case hcl.TraverseAttr:
		typeName = tt.Name
	default:
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action type is required.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	}

	switch tt := remain[1].(type) {
	case hcl.TraverseAttr:
		name = tt.Name
	default:
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action name is required.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	}

	remain = remain[2:]
	key := NoKey
	keyed := len(remain) > 0 && isInstanceKeyStep(remain[0])
	if keyed {
		var keyDiags tfdiags.Diagnostics
		key, keyDiags = parseInstanceKey(remain[0], keys, "action")
		diags = diags.Append(keyDiags)
		if keyDiags.HasErrors() {
			return AbsActionInstance{}, diags
		}
		remain = remain[1:]
	}

	switch {
	case len(remain) == 0:
		return moduleAddr.ActionInstance(typeName, name, key), diags
	case !keyed && len(remain) == 1:
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action instance key must be given in square brackets.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	case !keyed:
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	default:
		return AbsActionInstance{}, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	}
}
