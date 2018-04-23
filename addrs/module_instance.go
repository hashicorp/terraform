package addrs

import (
	"bytes"
	"fmt"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
)

// ModuleInstance is an address for a particular module instance within the
// dynamic module tree. This is an extension of the static traversals
// represented by type Module that deals with the possibility of a single
// module call producing multiple instances via the "count" and "for_each"
// arguments.
//
// Although ModuleInstance is a slice, it should be treated as immutable after
// creation.
type ModuleInstance []ModuleInstanceStep

func ParseModuleInstance(traversal hcl.Traversal) (ModuleInstance, tfdiags.Diagnostics) {
	mi, remain, diags := parseModuleInstancePrefix(traversal)
	if len(remain) != 0 {
		if len(remain) == len(traversal) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module instance address",
				Detail:   "A module instance address must begin with \"module.\".",
				Subject:  remain.SourceRange().Ptr(),
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module instance address",
				Detail:   "The module instance address is followed by additional invalid content.",
				Subject:  remain.SourceRange().Ptr(),
			})
		}
	}
	return mi, diags
}

func parseModuleInstancePrefix(traversal hcl.Traversal) (ModuleInstance, hcl.Traversal, tfdiags.Diagnostics) {
	remain := traversal
	var mi ModuleInstance
	var diags tfdiags.Diagnostics

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
			break
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
			break
		}
		remain = remain[1:]
		step := ModuleInstanceStep{
			Name: moduleName,
		}

		if len(remain) > 0 {
			if idx, ok := remain[0].(hcl.TraverseIndex); ok {
				remain = remain[1:]

				switch idx.Key.Type() {
				case cty.String:
					step.InstanceKey = StringKey(idx.Key.AsString())
				case cty.Number:
					var idxInt int
					err := gocty.FromCtyValue(idx.Key, &idxInt)
					if err == nil {
						step.InstanceKey = IntKey(idxInt)
					} else {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid address operator",
							Detail:   fmt.Sprintf("Invalid module index: %s.", err),
							Subject:  idx.SourceRange().Ptr(),
						})
					}
				default:
					// Should never happen, because no other types are allowed in traversal indices.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid address operator",
						Detail:   "Invalid module key: must be either a string or an integer.",
						Subject:  idx.SourceRange().Ptr(),
					})
				}
			}
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

// ModuleInstanceStep is a single traversal step through the dynamic module
// tree. It is used only as part of ModuleInstance.
type ModuleInstanceStep struct {
	Name        string
	InstanceKey InstanceKey
}

// RootModuleInstance is the module instance address representing the root
// module, which is also the zero value of ModuleInstance.
var RootModuleInstance ModuleInstance

// Child returns the address of a child module instance of the receiver,
// identified by the given name and key.
func (m ModuleInstance) Child(name string, key InstanceKey) ModuleInstance {
	ret := make(ModuleInstance, 0, len(m)+1)
	ret = append(ret, m...)
	return append(ret, ModuleInstanceStep{
		Name:        name,
		InstanceKey: key,
	})
}

// Parent returns the address of the parent module instance of the receiver, or
// the receiver itself if there is no parent (if it's the root module address).
func (m ModuleInstance) Parent() ModuleInstance {
	if len(m) == 0 {
		return m
	}
	return m[:len(m)-1]
}

// String returns a string representation of the receiver, in the format used
// within e.g. user-provided resource addresses.
//
// The address of the root module has the empty string as its representation.
func (m ModuleInstance) String() string {
	var buf bytes.Buffer
	sep := ""
	for _, step := range m {
		buf.WriteString(sep)
		buf.WriteString("module.")
		buf.WriteString(step.Name)
		if step.InstanceKey != NoKey {
			buf.WriteString(step.InstanceKey.String())
		}
		sep = "."
	}
	return buf.String()
}
