package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
)

// Reference describes a reference to an address with source location
// information.
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

		// A traversal starting with "module" can either be a reference to
		// an entire module instance or to a single output from a module
		// instance, depending on what we find after this introducer.

		callInstance := ModuleCallInstance{
			Call: ModuleCall{
				Name: callName,
			},
			Key: NoKey,
		}

		if len(remain) == 0 {
			// Reference to an entire module instance. Might alternatively
			// be a reference to a collection of instances of a particular
			// module, but the caller will need to deal with that ambiguity
			// since we don't have enough context here.
			return &Reference{
				Subject:     callInstance,
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
				Subject: ModuleCallOutput{
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
		// If it isn't a TraverseRoot then it must be a "data" reference.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `The "data" object does not support this operation.`,
			Subject:  traversal[0].SourceRange().Ptr(),
		})
		return nil, diags
	}

	attrTrav, ok := traversal[1].(hcl.TraverseAttr)
	if !ok {
		var what string
		switch mode {
		case DataResourceMode:
			what = "data source"
		default:
			what = "resource type"
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf(`A reference to a %s must be followed by at least one attribute access, specifying the resource name.`, what),
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
			Subject:     resourceInstAddr,
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
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
