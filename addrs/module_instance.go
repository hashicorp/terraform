package addrs

// ModuleInstance is an address for a particular module instance within the
// dynamic module tree. This is an extension of the static traversals
// represented by type Module that deals with the possibility of a single
// module call producing multiple instances via the "count" and "for_each"
// arguments.
//
// Although ModuleInstance is a slice, it should be treated as immutable after
// creation.
type ModuleInstance []ModuleInstanceStep

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
