// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ParseTargetAction attempts to interpret the given traversal as a targetable
// action address. The given traversal must be absolute, or this function will
// panic.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the Target value is invalid and
// must not be used.
//
// This function matches the behaviour of ParseTarget, except we are ensuring
// the caller is explicit about what kind of target they want to get. We prevent
// callers accidentally including action targets where they shouldn't be
// accessible by keeping these methods separate.
func ParseTargetAction(traversal hcl.Traversal) (*Target, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, false)
	if diags.HasErrors() {
		return nil, diags
	}

	if len(remain) == 0 {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action addresses must contain an action reference after the module reference.",
			Subject:  traversal.SourceRange().Ptr(),
		})
	}

	target, moreDiags := parseActionInstanceUnderModule(path, remain, tfdiags.SourceRangeFromHCL(traversal.SourceRange()))
	return target, diags.Append(moreDiags)
}

// ParseTargetActionStr is a helper wrapper around ParseTargetAction that takes
// a string and parses it into HCL before interpreting it.
//
// All the same cautions apply to this as with the equivalent ParseTargetStr.
func ParseTargetActionStr(str string) (*Target, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	target, targetDiags := ParseTargetAction(traversal)
	diags = diags.Append(targetDiags)
	return target, diags
}

func parseActionInstanceUnderModule(moduleAddr ModuleInstance, remain hcl.Traversal, srcRng tfdiags.SourceRange) (*Target, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if remain.RootName() != "action" {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action specification must start with `action`.",
			Subject:  remain.SourceRange().Ptr(),
		})
	}

	remain = remain[1:]

	if len(remain) < 2 {
		return nil, diags.Append(&hcl.Diagnostic{
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
		return nil, diags.Append(&hcl.Diagnostic{
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
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action name is required.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	}

	remain = remain[2:]
	switch len(remain) {
	case 0:
		return &Target{
			Subject:     moduleAddr.Action(typeName, name),
			SourceRange: srcRng,
		}, diags
	case 1:
		switch tt := remain[0].(type) {
		case hcl.TraverseIndex:
			key, err := ParseInstanceKey(tt.Key)
			if err != nil {
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid address",
					Detail:   fmt.Sprintf("Invalid action instance key: %s.", err),
					Subject:  remain[0].SourceRange().Ptr(),
				})
			}
			return &Target{
				Subject:     moduleAddr.ActionInstance(typeName, name, key),
				SourceRange: srcRng,
			}, diags
		case hcl.TraverseSplat:
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Action instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
		default:
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Action instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
		}
	default:
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	}
}
