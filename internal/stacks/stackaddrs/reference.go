// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

	case "provider":
		target, rng, remain, diags := parseProviderRef(traversal)
		ret.Target = target
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)
		return ret, remain, diags

	case "each", "count":
		attrName, rng, remain, diags := parseSingleAttrRef(traversal)
		if diags.HasErrors() {
			return ret, nil, diags
		}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)

		switch rootName {
		case "each":
			switch attrName {
			case "key":
				ret.Target = EachKey
				return ret, remain, diags
			case "value":
				ret.Target = EachValue
				return ret, remain, diags
			}
		case "count":
			switch attrName {
			case "index":
				ret.Target = CountIndex
				return ret, remain, diags
			}
		}
		// If we get here then rootName and attrName are not a valid combination.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unknown symbol",
			Detail:   fmt.Sprintf("The object %q has no attribute named %q.", rootName, attrName),
			Subject:  traversal[1].SourceRange().Ptr(),
		})
		return ret, nil, diags

	case "self":
		ret.Target = Self
		ret.SourceRange = tfdiags.SourceRangeFromHCL(traversal[0].SourceRange())
		return ret, traversal[1:], diags

	case "terraform":
		attrName, rng, remain, diags := parseSingleAttrRef(traversal)
		if diags.HasErrors() {
			return ret, nil, diags
		}
		ret.SourceRange = tfdiags.SourceRangeFromHCL(rng)

		switch attrName {
		case "applying":
			ret.Target = TerraformApplying
			return ret, remain, diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to unknown symbol",
				Detail:   fmt.Sprintf("The object %q has no attribute named %q.", rootName, attrName),
				Subject:  traversal[1].SourceRange().Ptr(),
			})
			return ret, remain, diags
		}

	case "_test_only_global":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		ret.Target = TestOnlyGlobal{Name: name}
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

func parseProviderRef(traversal hcl.Traversal) (ProviderConfigRef, hcl.Range, hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if len(traversal) < 3 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   "The \"provider\" symbol must be followed by two attribute access operations, selecting a provider type and a provider configuration name.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return ProviderConfigRef{}, hcl.Range{}, nil, diags
	}
	if typeTrav, ok := traversal[1].(hcl.TraverseAttr); ok {
		if nameTrav, ok := traversal[2].(hcl.TraverseAttr); ok {
			ret := ProviderConfigRef{
				ProviderLocalName: typeTrav.Name,
				Name:              nameTrav.Name,
			}
			return ret, traversal.SourceRange(), traversal[3:], diags
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "The \"provider\" object's attributes do not support this operation.",
				Subject:  traversal[1].SourceRange().Ptr(),
			})
		}
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   "The \"provider\" object does not support this operation.",
			Subject:  traversal[1].SourceRange().Ptr(),
		})
	}
	return ProviderConfigRef{}, hcl.Range{}, nil, diags
}

func (r Reference) Absolute(stack StackInstance) AbsReference {
	return AbsReference{
		Stack: stack,
		Ref:   r,
	}
}

// AbsReference is an absolute form of [Reference] that is to be resolved
// in the global scope of a particular stack.
//
// It's not meaningful to use this type for references to objects that exist
// only in a more specific scope, such as each.key, each.value, etc, because
// those would require additional information about exactly which object
// they are being resolved in terms of.
type AbsReference struct {
	Stack StackInstance
	Ref   Reference
}

func (r AbsReference) Target() AbsReferenceable {
	return AbsReferenceable{
		Stack: r.Stack,
		Item:  r.Ref.Target,
	}
}

func (r AbsReference) SourceRange() tfdiags.SourceRange {
	return r.Ref.SourceRange
}
