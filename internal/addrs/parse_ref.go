// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Reference describes a reference to an address with source location
// information.
type Reference struct {
	Subject     Referenceable
	SourceRange tfdiags.SourceRange
	Remaining   hcl.Traversal
}

// DisplayString returns a string that approximates the subject and remaining
// traversal of the reciever in a way that resembles the Terraform language
// syntax that could've produced it.
//
// It's not guaranteed to actually be a valid Terraform language expression,
// since the intended use here is primarily for UI messages such as
// diagnostics.
func (r *Reference) DisplayString() string {
	if len(r.Remaining) == 0 {
		// Easy case: we can just return the subject's string.
		return r.Subject.String()
	}

	var ret strings.Builder
	ret.WriteString(r.Subject.String())
	for _, step := range r.Remaining {
		switch tStep := step.(type) {
		case hcl.TraverseRoot:
			ret.WriteString(tStep.Name)
		case hcl.TraverseAttr:
			ret.WriteByte('.')
			ret.WriteString(tStep.Name)
		case hcl.TraverseIndex:
			ret.WriteByte('[')
			switch tStep.Key.Type() {
			case cty.String:
				ret.WriteString(fmt.Sprintf("%q", tStep.Key.AsString()))
			case cty.Number:
				bf := tStep.Key.AsBigFloat()
				ret.WriteString(bf.Text('g', 10))
			}
			ret.WriteByte(']')
		}
	}
	return ret.String()
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

// ParseRefFromTestingScope adds check blocks and outputs into the available
// references returned by ParseRef.
//
// The testing files and functionality have a slightly expanded referencing
// scope and so should use this function to retrieve references.
func ParseRefFromTestingScope(traversal hcl.Traversal) (*Reference, tfdiags.Diagnostics) {
	root := traversal.RootName()
	rootRange := traversal[0].SourceRange()

	var diags tfdiags.Diagnostics
	var reference *Reference

	switch root {
	case "output":
		name, rng, remain, outputDiags := parseSingleAttrRef(traversal)
		reference = &Reference{
			Subject:     OutputValue{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}
		diags = outputDiags
	case "check":
		name, rng, remain, checkDiags := parseSingleAttrRef(traversal)
		reference = &Reference{
			Subject:     Check{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}
		diags = checkDiags
	case "run":
		name, rng, remain, runDiags := parseSingleAttrRef(traversal)
		reference = &Reference{
			Subject:     Run{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}
		diags = runDiags
	case "plan", "state":
		// These names are all pre-emptively reserved in the hope of landing
		// some version of referencing the plan and state files in test
		// assertions.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved symbol name",
			Detail:   fmt.Sprintf("The symbol name %q is reserved for use in a future Terraform version. If you are using a provider that already uses this as a resource type name, add the prefix \"resource.\" to force interpretation as a resource type name.", root),
			Subject:  rootRange.Ptr(),
		})
		return nil, diags
	}

	if reference != nil {
		if len(reference.Remaining) == 0 {
			reference.Remaining = nil
		}
		return reference, diags
	}

	// If it's not an output or a check block, then just parse it as normal.
	return ParseRef(traversal)
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

// ParseRefStrFromTestingScope matches ParseRefStr except it supports the
// references supported by ParseRefFromTestingScope.
func ParseRefStrFromTestingScope(str string) (*Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	ref, targetDiags := ParseRefFromTestingScope(traversal)
	diags = diags.Append(targetDiags)
	return ref, diags
}

func parseRef(traversal hcl.Traversal) (*Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	root := traversal.RootName()
	rootRange := traversal[0].SourceRange()

	switch root {

	case "count":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     CountAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "each":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     ForEachAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "data":
		if len(traversal) < 3 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "data" object must be followed by two attribute names: the data source type and the resource name.`,
				Subject:  traversal.SourceRange().Ptr(),
			})
			return nil, diags
		}
		remain := traversal[1:] // trim off "data" so we can use our shared resource reference parser
		return parseResourceRef(DataResourceMode, rootRange, remain)

	case "resource":
		// This is an alias for the normal case of just using a managed resource
		// type as a top-level symbol, which will serve as an escape mechanism
		// if a later edition of the Terraform language introduces a new
		// reference prefix that conflicts with a resource type name in an
		// existing provider. In that case, the edition upgrade tool can
		// rewrite foo.bar into resource.foo.bar to ensure that "foo" remains
		// interpreted as a resource type name rather than as the new reserved
		// word.
		if len(traversal) < 3 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "resource" object must be followed by two attribute names: the resource type and the resource name.`,
				Subject:  traversal.SourceRange().Ptr(),
			})
			return nil, diags
		}
		remain := traversal[1:] // trim off "resource" so we can use our shared resource reference parser
		return parseResourceRef(ManagedResourceMode, rootRange, remain)

	case "ephemeral":
		if len(traversal) < 3 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "ephemeral" object must be followed by two attribute names: the ephemeral resource type and the resource name.`,
				Subject:  traversal.SourceRange().Ptr(),
			})
			return nil, diags
		}
		remain := traversal[1:] // trim off "ephemeral" so we can use our shared resource reference parser
		return parseResourceRef(EphemeralResourceMode, rootRange, remain)

	case "local":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     LocalValue{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "module":
		callName, callRange, remain, diags := parseSingleAttrRef(traversal)
		if diags.HasErrors() {
			return nil, diags
		}

		// A traversal starting with "module" can either be a reference to an
		// entire module, or to a single output from a module instance,
		// depending on what we find after this introducer.
		callInstance := ModuleCallInstance{
			Call: ModuleCall{
				Name: callName,
			},
			Key: NoKey,
		}

		if len(remain) == 0 {
			// Reference to an entire module. Might alternatively be a
			// reference to a single instance of a particular module, but the
			// caller will need to deal with that ambiguity since we don't have
			// enough context here.
			return &Reference{
				Subject:     callInstance.Call,
				SourceRange: tfdiags.SourceRangeFromHCL(callRange),
				Remaining:   remain,
			}, diags
		}

		if idxTrav, ok := remain[0].(hcl.TraverseIndex); ok {
			var err error
			callInstance.Key, err = ParseInstanceKey(idxTrav.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid index key",
					Detail:   fmt.Sprintf("Invalid index for module instance: %s.", err),
					Subject:  &idxTrav.SrcRange,
				})
				return nil, diags
			}
			remain = remain[1:]

			if len(remain) == 0 {
				// Also a reference to an entire module instance, but we have a key
				// now.
				return &Reference{
					Subject:     callInstance,
					SourceRange: tfdiags.SourceRangeFromHCL(hcl.RangeBetween(callRange, idxTrav.SrcRange)),
					Remaining:   remain,
				}, diags
			}
		}

		if attrTrav, ok := remain[0].(hcl.TraverseAttr); ok {
			remain = remain[1:]
			return &Reference{
				Subject: ModuleCallInstanceOutput{
					Name: attrTrav.Name,
					Call: callInstance,
				},
				SourceRange: tfdiags.SourceRangeFromHCL(hcl.RangeBetween(callRange, attrTrav.SrcRange)),
				Remaining:   remain,
			}, diags
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   "Module instance objects do not support this operation.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return nil, diags

	case "path":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     PathAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "self":
		return &Reference{
			Subject:     Self,
			SourceRange: tfdiags.SourceRangeFromHCL(rootRange),
			Remaining:   traversal[1:],
		}, diags

	case "terraform":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     TerraformAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "var":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &Reference{
			Subject:     InputVariable{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "template", "lazy", "arg":
		// These names are all pre-emptively reserved in the hope of landing
		// some version of "template values" or "lazy expressions" feature
		// before the next opt-in language edition, but don't yet do anything.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reserved symbol name",
			Detail:   fmt.Sprintf("The symbol name %q is reserved for use in a future Terraform version. If you are using a provider that already uses this as a resource type name, add the prefix \"resource.\" to force interpretation as a resource type name.", root),
			Subject:  rootRange.Ptr(),
		})
		return nil, diags

	default:
		return parseResourceRef(ManagedResourceMode, rootRange, traversal)
	}
}

func parseResourceRef(mode ResourceMode, startRange hcl.Range, traversal hcl.Traversal) (*Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if len(traversal) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `A reference to a resource type must be followed by at least one attribute access, specifying the resource name.`,
			Subject:  hcl.RangeBetween(traversal[0].SourceRange(), traversal[len(traversal)-1].SourceRange()).Ptr(),
		})
		return nil, diags
	}

	var typeName, name string
	switch tt := traversal[0].(type) { // Could be either root or attr, depending on our resource mode
	case hcl.TraverseRoot:
		typeName = tt.Name
	case hcl.TraverseAttr:
		typeName = tt.Name
	default:
		switch mode {
		case ManagedResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "resource" object does not support this operation.`,
				Subject:  traversal[0].SourceRange().Ptr(),
			})
		case DataResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "data" object does not support this operation.`,
				Subject:  traversal[0].SourceRange().Ptr(),
			})
		case EphemeralResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The "ephemeral" object does not support this operation.`,
				Subject:  traversal[0].SourceRange().Ptr(),
			})
		default:
			// Shouldn't get here because the above should be exhaustive for
			// all of the resource modes. But we'll still return a
			// minimally-passable error message so that the won't totally
			// misbehave if we forget to update this in future.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   `The left operand does not support this operation.`,
				Subject:  traversal[0].SourceRange().Ptr(),
			})
		}
		return nil, diags
	}

	attrTrav, ok := traversal[1].(hcl.TraverseAttr)
	if !ok {
		var what string
		switch mode {
		case DataResourceMode:
			what = "a data source"
		case EphemeralResourceMode:
			what = "an ephemeral resource type"
		default:
			what = "a resource type"
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf(`A reference to %s must be followed by at least one attribute access, specifying the resource name.`, what),
			Subject:  traversal[1].SourceRange().Ptr(),
		})
		return nil, diags
	}
	name = attrTrav.Name
	rng := hcl.RangeBetween(startRange, attrTrav.SrcRange)
	remain := traversal[2:]

	resourceAddr := Resource{
		Mode: mode,
		Type: typeName,
		Name: name,
	}
	resourceInstAddr := ResourceInstance{
		Resource: resourceAddr,
		Key:      NoKey,
	}

	if len(remain) == 0 {
		// This might actually be a reference to the collection of all instances
		// of the resource, but we don't have enough context here to decide
		// so we'll let the caller resolve that ambiguity.
		return &Reference{
			Subject:     resourceAddr,
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
		}, diags
	}

	if idxTrav, ok := remain[0].(hcl.TraverseIndex); ok {
		var err error
		resourceInstAddr.Key, err = ParseInstanceKey(idxTrav.Key)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index key",
				Detail:   fmt.Sprintf("Invalid index for resource instance: %s.", err),
				Subject:  &idxTrav.SrcRange,
			})
			return nil, diags
		}
		remain = remain[1:]
		rng = hcl.RangeBetween(rng, idxTrav.SrcRange)
	}

	return &Reference{
		Subject:     resourceInstAddr,
		SourceRange: tfdiags.SourceRangeFromHCL(rng),
		Remaining:   remain,
	}, diags
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
