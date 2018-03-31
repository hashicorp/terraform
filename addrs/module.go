package addrs

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
