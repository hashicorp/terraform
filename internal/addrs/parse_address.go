// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// instanceKeys describes which instance keys are valid in an address being
// parsed from a traversal.
type instanceKeys int

const (
	// noInstanceKeys is for configuration addresses, which can't have any
	// instance keys.
	noInstanceKeys instanceKeys = iota

	// knownInstanceKeys is for concrete addresses, where every instance key
	// must be known.
	knownInstanceKeys

	// wildcardInstanceKeys is for patterns and partial-expanded addresses,
	// where any instance key can also be a wildcard. A wildcard is written as
	// [*] in a traversal pattern, or is an index with an unknown value, and
	// is parsed as WildcardKey.
	wildcardInstanceKeys
)

// parseModuleInstancePrefix parses the module path at the start of the given
// traversal, with instance keys as allowed by keys.
//
// The remainder of the traversal is returned for the caller to interpret as
// an object within the module. It always starts with a TraverseRoot if it's
// not empty.
func parseModuleInstancePrefix(traversal hcl.Traversal, keys instanceKeys) (ModuleInstance, hcl.Traversal, tfdiags.Diagnostics) {
	remain := traversal
	var mi ModuleInstance
	var diags tfdiags.Diagnostics

LOOP:
	for len(remain) > 0 {
		var next string
		switch tt := remain[0].(type) {
		case hcl.TraverseRoot:
			next = tt.Name
		case hcl.TraverseAttr:
			next = tt.Name
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address operator",
				Detail:   "Module address prefix must be followed by dot and then a name.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			break LOOP
		}

		if next != "module" {
			break
		}

		kwRange := remain[0].SourceRange()
		remain = remain[1:]
		// If we have the prefix "module" then we should be followed by an
		// module call name, as an attribute, and then optionally an index step
		// giving the instance key.
		if len(remain) == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address operator",
				Detail:   "Prefix \"module.\" must be followed by a module name.",
				Subject:  &kwRange,
			})
			break
		}

		var moduleName string
		switch tt := remain[0].(type) {
		case hcl.TraverseAttr:
			moduleName = tt.Name
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address operator",
				Detail:   "Prefix \"module.\" must be followed by a module name.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			break LOOP
		}
		remain = remain[1:]
		step := ModuleInstanceStep{
			Name: moduleName,
		}

		if len(remain) > 0 && isInstanceKeyStep(remain[0]) {
			key, keyDiags := parseInstanceKey(remain[0], keys, "module")
			diags = diags.Append(keyDiags)
			step.InstanceKey = key
			remain = remain[1:]
		}

		mi = append(mi, step)
	}

	var retRemain hcl.Traversal
	if len(remain) > 0 {
		retRemain = make(hcl.Traversal, len(remain))
		copy(retRemain, remain)
		// The first element here might be either a TraverseRoot or a
		// TraverseAttr, depending on whether we had a module address on the
		// front. To make life easier for callers, we'll normalize to always
		// start with a TraverseRoot.
		if tt, ok := retRemain[0].(hcl.TraverseAttr); ok {
			retRemain[0] = hcl.TraverseRoot{
				Name:     tt.Name,
				SrcRange: tt.SrcRange,
			}
		}
	}

	return mi, retRemain, diags
}

// parseResourceUnderModule parses the resource address at the start of the
// given traversal, which is the remainder after parsing its module path, and
// its instance key if it has one, as allowed by keys.
//
// If allowExtra is set, the traversal after the resource address is returned
// for the caller to interpret. Otherwise there must be nothing else in the
// traversal.
func parseResourceUnderModule(remain hcl.Traversal, keys instanceKeys, allowExtra bool) (Resource, InstanceKey, hcl.Traversal, tfdiags.Diagnostics) {
	// Note that this helper is used for all of the address types which
	// include a resource, such as targets, move endpoints, and removed blocks,
	// so its error messages should be generic enough to suit all of them.

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
		return Resource{}, NoKey, nil, diags
	}

	if len(remain) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Resource specification must include a resource type and name.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return Resource{}, NoKey, nil, diags
	}

	var typeName, name string
	switch tt := remain[0].(type) {
	case hcl.TraverseRoot:
		typeName = tt.Name
	case hcl.TraverseAttr:
		typeName = tt.Name
	default:
		var detail string
		switch mode {
		case ManagedResourceMode:
			detail = "A resource type name is required."
		case DataResourceMode:
			detail = "A data source name is required."
		case EphemeralResourceMode:
			detail = "An ephemeral resource type name is required."
		case ListResourceMode:
			detail = "A list resource type name is required."
		default:
			panic("unknown mode")
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   detail,
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return Resource{}, NoKey, nil, diags
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
		return Resource{}, NoKey, nil, diags
	}

	resource := Resource{
		Mode: mode,
		Type: typeName,
		Name: name,
	}
	remain = remain[2:]

	key := NoKey
	keyed := len(remain) > 0 && isInstanceKeyStep(remain[0])
	if keyed {
		var keyDiags tfdiags.Diagnostics
		key, keyDiags = parseInstanceKey(remain[0], keys, "resource")
		diags = diags.Append(keyDiags)
		if keyDiags.HasErrors() {
			return Resource{}, NoKey, nil, diags
		}
		remain = remain[1:]
	}

	if allowExtra || len(remain) == 0 {
		return resource, key, remain, diags
	}

	switch {
	case !keyed && keys != noInstanceKeys && len(remain) == 1:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Resource instance key must be given in square brackets.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	case !keyed && keys != noInstanceKeys:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
	}
	return Resource{}, NoKey, nil, diags
}

// isInstanceKeyStep returns true if the given traversal step can represent an
// instance key.
func isInstanceKeyStep(step hcl.Traverser) bool {
	switch step.(type) {
	case hcl.TraverseIndex, hcl.TraverseSplat:
		return true
	default:
		return false
	}
}

// parseInstanceKey parses the given index or splat traversal step as the
// instance key of an object of the given kind, which is used only in error
// messages, as allowed by keys.
func parseInstanceKey(step hcl.Traverser, keys instanceKeys, kind string) (InstanceKey, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rng := step.SourceRange().Ptr()

	if keys == noInstanceKeys {
		switch kind {
		case "module":
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Module instance keys not allowed",
				Detail:   "Module address must be a module (e.g. \"module.foo\"), not a module instance (e.g. \"module.foo[1]\").",
				Subject:  rng,
			})
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Resource instance keys not allowed",
				Detail:   "Resource address must be a resource (e.g. \"test_instance.foo\"), not a resource instance (e.g. \"test_instance.foo[1]\").",
				Subject:  rng,
			})
		}
		return NoKey, diags
	}

	idx, isIndex := step.(hcl.TraverseIndex)
	if !isIndex || !idx.Key.IsKnown() {
		if keys == wildcardInstanceKeys {
			return WildcardKey, diags
		}

		detail := fmt.Sprintf("The %s instance key cannot be a wildcard in this address.", kind)
		if isIndex {
			detail = fmt.Sprintf("The %s instance key must be known in this address.", kind)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   detail,
			Subject:  rng,
		})
		return NoKey, diags
	}

	key, err := ParseInstanceKey(idx.Key)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   fmt.Sprintf("Invalid %s instance key: %s.", kind, err),
			Subject:  rng,
		})
		return NoKey, diags
	}
	return key, diags
}
