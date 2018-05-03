package addrs

import (
	"fmt"
)

// ModuleCall is the address of a call from the current module to a child
// module.
//
// There is no "Abs" version of ModuleCall because an absolute module path
// is represented by ModuleInstance.
type ModuleCall struct {
	referenceable
	Name string
}

func (c ModuleCall) String() string {
	return "module." + c.Name
}

// Instance returns the address of an instance of the receiver identified by
// the given key.
func (c ModuleCall) Instance(key InstanceKey) ModuleCallInstance {
	return ModuleCallInstance{
		Call: c,
		Key:  key,
	}
}

// ModuleCallInstance is the address of one instance of a module created from
// a module call, which might create multiple instances using "count" or
// "for_each" arguments.
type ModuleCallInstance struct {
	referenceable
	Call ModuleCall
	Key  InstanceKey
}

func (c ModuleCallInstance) String() string {
	if c.Key == NoKey {
		return c.Call.String()
	}
	return fmt.Sprintf("module.%s%s", c.Call.Name, c.Key)
}

// ModuleInstance returns the address of the module instance that corresponds
// to the receiving call instance when resolved in the given calling module.
// In other words, it returns the child module instance that the receving
// call instance creates.
func (c ModuleCallInstance) ModuleInstance(caller ModuleInstance) ModuleInstance {
	return caller.Child(c.Call.Name, c.Key)
}

// Output returns the address of an output of the receiver identified by its
// name.
func (c ModuleCallInstance) Output(name string) ModuleCallOutput {
	return ModuleCallOutput{
		Call: c,
		Name: name,
	}
}

// ModuleCallOutput is the address of a particular named output produced by
// an instance of a module call.
type ModuleCallOutput struct {
	referenceable
	Call ModuleCallInstance
	Name string
}

func (co ModuleCallOutput) String() string {
	return fmt.Sprintf("%s.%s", co.Call.String(), co.Name)
}

// AbsOutputValue returns the absolute output value address that corresponds
// to the receving module call output address, once resolved in the given
// calling module.
func (co ModuleCallOutput) AbsOutputValue(caller ModuleInstance) AbsOutputValue {
	moduleAddr := co.Call.ModuleInstance(caller)
	return moduleAddr.OutputValue(co.Name)
}
