// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/internal/tfdiags"
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

var (
	_ Targetable = ModuleInstance(nil)
)

func ParseModuleInstance(traversal hcl.Traversal) (ModuleInstance, tfdiags.Diagnostics) {
	mi, remain, diags := parseModuleInstancePrefix(traversal, false)
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

// ParseModuleInstanceStr is a helper wrapper around ParseModuleInstance
// that takes a string and parses it with the HCL native syntax traversal parser
// before interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a reference string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseModuleInstance.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// then the returned address is invalid.
func ParseModuleInstanceStr(str string) (ModuleInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	addr, addrDiags := ParseModuleInstance(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

func parseModuleInstancePrefix(traversal hcl.Traversal, allowPartial bool) (ModuleInstance, hcl.Traversal, tfdiags.Diagnostics) {
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

		if len(remain) > 0 {
			switch idx := remain[0].(type) {
			case hcl.TraverseIndex:
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

			case hcl.TraverseSplat:
				if allowPartial {
					remain = remain[1:]
					step.InstanceKey = WildcardKey
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

// UnkeyedInstanceShim is a shim method for converting a Module address to the
// equivalent ModuleInstance address that assumes that no modules have
// keyed instances.
//
// This is a temporary allowance for the fact that Terraform does not presently
// support "count" and "for_each" on modules, and thus graph building code that
// derives graph nodes from configuration must just assume unkeyed modules
// in order to construct the graph. At a later time when "count" and "for_each"
// support is added for modules, all callers of this method will need to be
// reworked to allow for keyed module instances.
func (m Module) UnkeyedInstanceShim() ModuleInstance {
	path := make(ModuleInstance, len(m))
	for i, name := range m {
		path[i] = ModuleInstanceStep{Name: name}
	}
	return path
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

// IsRoot returns true if the receiver is the address of the root module instance,
// or false otherwise.
func (m ModuleInstance) IsRoot() bool {
	return len(m) == 0
}

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

// ChildCall returns the address of a module call within the receiver,
// identified by the given name.
func (m ModuleInstance) ChildCall(name string) AbsModuleCall {
	return AbsModuleCall{
		Module: m,
		Call:   ModuleCall{Name: name},
	}
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
	if len(m) == 0 {
		return ""
	}
	// Calculate minimal necessary space (no instance keys).
	l := 0
	for _, step := range m {
		l += len(step.Name)
	}
	buf := strings.Builder{}
	// 8 is len(".module.") which separates entries.
	buf.Grow(l + len(m)*8)
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

type moduleInstanceKey string

func (m ModuleInstance) UniqueKey() UniqueKey {
	return moduleInstanceKey(m.String())
}

func (mk moduleInstanceKey) uniqueKeySigil() {}

// Equal returns true if the receiver and the given other value
// contains the exact same parts.
func (m ModuleInstance) Equal(o ModuleInstance) bool {
	if len(m) != len(o) {
		return false
	}

	for i := range m {
		if m[i] != o[i] {
			return false
		}
	}
	return true
}

// Less returns true if the receiver should sort before the given other value
// in a sorted list of addresses.
func (m ModuleInstance) Less(o ModuleInstance) bool {
	if len(m) != len(o) {
		// Shorter path sorts first.
		return len(m) < len(o)
	}

	for i := range m {
		mS, oS := m[i], o[i]
		switch {
		case mS.Name != oS.Name:
			return mS.Name < oS.Name
		case mS.InstanceKey != oS.InstanceKey:
			return InstanceKeyLess(mS.InstanceKey, oS.InstanceKey)
		}
	}

	return false
}

// Ancestors returns a slice containing the receiver and all of its ancestor
// module instances, all the way up to (and including) the root module.
// The result is ordered by depth, with the root module always first.
//
// Since the result always includes the root module, a caller may choose to
// ignore it by slicing the result with [1:].
func (m ModuleInstance) Ancestors() []ModuleInstance {
	ret := make([]ModuleInstance, 0, len(m)+1)
	for i := 0; i <= len(m); i++ {
		ret = append(ret, m[:i])
	}
	return ret
}

// IsAncestor returns true if the receiver is an ancestor of the given
// other value.
func (m ModuleInstance) IsAncestor(o ModuleInstance) bool {
	// Longer or equal sized paths means the receiver cannot
	// be an ancestor of the given module insatnce.
	if len(m) >= len(o) {
		return false
	}

	for i, ms := range m {
		if ms.Name != o[i].Name {
			return false
		}
		if ms.InstanceKey != NoKey && ms.InstanceKey != o[i].InstanceKey {
			return false
		}
	}

	return true
}

// Call returns the module call address that corresponds to the given module
// instance, along with the address of the module instance that contains it.
//
// There is no call for the root module, so this method will panic if called
// on the root module address.
//
// A single module call can produce potentially many module instances, so the
// result discards any instance key that might be present on the last step
// of the instance. To retain this, use CallInstance instead.
//
// In practice, this just turns the last element of the receiver into a
// ModuleCall and then returns a slice of the receiever that excludes that
// last part. This is just a convenience for situations where a call address
// is required, such as when dealing with *Reference and Referencable values.
func (m ModuleInstance) Call() (ModuleInstance, ModuleCall) {
	if len(m) == 0 {
		panic("cannot produce ModuleCall for root module")
	}

	inst, lastStep := m[:len(m)-1], m[len(m)-1]
	return inst, ModuleCall{
		Name: lastStep.Name,
	}
}

// AbsCall returns the same information as [ModuleInstance.Call], but returns
// it as a single [AbsModuleCall] value rather than the containing module
// and the local call address separately.
func (m ModuleInstance) AbsCall() AbsModuleCall {
	container, call := m.Call()
	return AbsModuleCall{
		Module: container,
		Call:   call,
	}
}

// CallInstance returns the module call instance address that corresponds to
// the given module instance, along with the address of the module instance
// that contains it.
//
// There is no call for the root module, so this method will panic if called
// on the root module address.
//
// In practice, this just turns the last element of the receiver into a
// ModuleCallInstance and then returns a slice of the receiever that excludes
// that last part. This is just a convenience for situations where a call\
// address is required, such as when dealing with *Reference and Referencable
// values.
func (m ModuleInstance) CallInstance() (ModuleInstance, ModuleCallInstance) {
	if len(m) == 0 {
		panic("cannot produce ModuleCallInstance for root module")
	}

	inst, lastStep := m[:len(m)-1], m[len(m)-1]
	return inst, ModuleCallInstance{
		Call: ModuleCall{
			Name: lastStep.Name,
		},
		Key: lastStep.InstanceKey,
	}
}

// TargetContains implements Targetable by returning true if the given other
// address either matches the receiver, is a sub-module-instance of the
// receiver, or is a targetable absolute address within a module that
// is contained within the reciever.
func (m ModuleInstance) TargetContains(other Targetable) bool {
	switch to := other.(type) {
	case Module:
		if len(to) < len(m) {
			// Can't be contained if the path is shorter
			return false
		}
		// Other is contained if its steps match for the length of our own path.
		for i, ourStep := range m {
			otherStep := to[i]

			// We can't contain an entire module if we have a specific instance
			// key. The case of NoKey is OK because this address is either
			// meant to address an unexpanded module, or a single instance of
			// that module, and both of those are a covered in-full by the
			// Module address.
			if ourStep.InstanceKey != NoKey {
				return false
			}

			if ourStep.Name != otherStep {
				return false
			}
		}
		// If we fall out here then the prefixed matched, so it's contained.
		return true

	case ModuleInstance:
		if len(to) < len(m) {
			return false
		}
		for i, ourStep := range m {
			otherStep := to[i]

			if ourStep.Name != otherStep.Name {
				return false
			}

			// if this is our last step, because all targets are parsed as
			// instances, this may be a ModuleInstance intended to be used as a
			// Module.
			if i == len(m)-1 {
				if ourStep.InstanceKey == NoKey {
					// If the other step is a keyed instance, then we contain that
					// step, and if it isn't it's a match, which is true either way
					return true
				}
			}

			if ourStep.InstanceKey != otherStep.InstanceKey {
				return false
			}

		}
		return true

	case ConfigResource:
		return m.TargetContains(to.Module)

	case AbsResource:
		return m.TargetContains(to.Module)

	case AbsResourceInstance:
		return m.TargetContains(to.Module)

	default:
		return false
	}
}

// Module returns the address of the module that this instance is an instance
// of.
func (m ModuleInstance) Module() Module {
	if len(m) == 0 {
		return nil
	}
	ret := make(Module, len(m))
	for i, step := range m {
		ret[i] = step.Name
	}
	return ret
}

// ContainingModule returns the address of the module instance as if the last
// step wasn't instanced. For example, it turns module.child[0] into
// module.child and module[0].child[0] into module[0].child.
func (m ModuleInstance) ContainingModule() ModuleInstance {
	if len(m) == 0 {
		return nil
	}

	ret := m.Parent()
	return ret.Child(m[len(m)-1].Name, NoKey)
}

func (m ModuleInstance) AddrType() TargetableAddrType {
	return ModuleInstanceAddrType
}

func (m ModuleInstance) targetableSigil() {
	// ModuleInstance is targetable
}

func (m ModuleInstance) absMoveableSigil() {
	// ModuleInstance is moveable
}

// IsDeclaredByCall returns true if the receiver is an instance of the given
// AbsModuleCall.
func (m ModuleInstance) IsDeclaredByCall(other AbsModuleCall) bool {
	// Compare len(m) to len(other.Module+1) because the final module instance
	// step in other is stored in the AbsModuleCall.Call
	if len(m) > len(other.Module)+1 || len(m) == 0 && len(other.Module) == 0 {
		return false
	}

	// Verify that the other's ModuleInstance matches the receiver.
	inst, lastStep := other.Module, other.Call
	for i := range inst {
		if inst[i] != m[i] {
			return false
		}
	}

	// Now compare the final step of the received with the other Call, where
	// only the name needs to match.
	return lastStep.Name == m[len(m)-1].Name
}

func (s ModuleInstanceStep) String() string {
	if s.InstanceKey != NoKey {
		return s.Name + s.InstanceKey.String()
	}
	return s.Name
}
