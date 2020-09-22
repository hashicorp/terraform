package addrs

import (
	"strings"
)

// Module is an address for a module call within configuration. This is
// the static counterpart of ModuleInstance, representing a traversal through
// the static module call tree in configuration and does not take into account
// the potentially-multiple instances of a module that might be created by
// "count" and "for_each" arguments within those calls.
//
// This type should be used only in very specialized cases when working with
// the static module call tree. Type ModuleInstance is appropriate in more cases.
//
// Although Module is a slice, it should be treated as immutable after creation.
type Module []string

// RootModule is the module address representing the root of the static module
// call tree, which is also the zero value of Module.
//
// Note that this is not the root of the dynamic module tree, which is instead
// represented by RootModuleInstance.
var RootModule Module

// IsRoot returns true if the receiver is the address of the root module,
// or false otherwise.
func (m Module) IsRoot() bool {
	return len(m) == 0
}

func (m Module) String() string {
	if len(m) == 0 {
		return ""
	}
	var steps []string
	for _, s := range m {
		steps = append(steps, "module", s)
	}
	return strings.Join(steps, ".")
}

func (m Module) Equal(other Module) bool {
	return m.String() == other.String()
}

func (m Module) targetableSigil() {
	// Module is targetable
}

// TargetContains implements Targetable for Module by returning true if the given other
// address either matches the receiver, is a sub-module-instance of the
// receiver, or is a targetable absolute address within a module that
// is contained within the receiver.
func (m Module) TargetContains(other Targetable) bool {
	switch to := other.(type) {

	case Module:
		if len(to) < len(m) {
			// Can't be contained if the path is shorter
			return false
		}
		// Other is contained if its steps match for the length of our own path.
		for i, ourStep := range m {
			otherStep := to[i]
			if ourStep != otherStep {
				return false
			}
		}
		// If we fall out here then the prefixed matched, so it's contained.
		return true

	case ModuleInstance:
		return m.TargetContains(to.Module())

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

// Child returns the address of a child call in the receiver, identified by the
// given name.
func (m Module) Child(name string) Module {
	ret := make(Module, 0, len(m)+1)
	ret = append(ret, m...)
	return append(ret, name)
}

// Parent returns the address of the parent module of the receiver, or the
// receiver itself if there is no parent (if it's the root module address).
func (m Module) Parent() Module {
	if len(m) == 0 {
		return m
	}
	return m[:len(m)-1]
}

// Call returns the module call address that corresponds to the given module
// instance, along with the address of the module that contains it.
//
// There is no call for the root module, so this method will panic if called
// on the root module address.
//
// In practice, this just turns the last element of the receiver into a
// ModuleCall and then returns a slice of the receiever that excludes that
// last part. This is just a convenience for situations where a call address
// is required, such as when dealing with *Reference and Referencable values.
func (m Module) Call() (Module, ModuleCall) {
	if len(m) == 0 {
		panic("cannot produce ModuleCall for root module")
	}

	caller, callName := m[:len(m)-1], m[len(m)-1]
	return caller, ModuleCall{
		Name: callName,
	}
}

// Ancestors returns a slice containing the receiver and all of its ancestor
// modules, all the way up to (and including) the root module.  The result is
// ordered by depth, with the root module always first.
//
// Since the result always includes the root module, a caller may choose to
// ignore it by slicing the result with [1:].
func (m Module) Ancestors() []Module {
	ret := make([]Module, 0, len(m)+1)
	for i := 0; i <= len(m); i++ {
		ret = append(ret, m[:i])
	}
	return ret
}
