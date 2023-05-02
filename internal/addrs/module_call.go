// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
)

// ModuleCall is the address of a call from the current module to a child
// module.
type ModuleCall struct {
	referenceable
	Name string
}

func (c ModuleCall) String() string {
	return "module." + c.Name
}

func (c ModuleCall) UniqueKey() UniqueKey {
	return c // A ModuleCall is its own UniqueKey
}

func (c ModuleCall) uniqueKeySigil() {}

// Instance returns the address of an instance of the receiver identified by
// the given key.
func (c ModuleCall) Instance(key InstanceKey) ModuleCallInstance {
	return ModuleCallInstance{
		Call: c,
		Key:  key,
	}
}

func (c ModuleCall) Absolute(moduleAddr ModuleInstance) AbsModuleCall {
	return AbsModuleCall{
		Module: moduleAddr,
		Call:   c,
	}
}

func (c ModuleCall) Equal(other ModuleCall) bool {
	return c.Name == other.Name
}

// AbsModuleCall is the address of a "module" block relative to the root
// of the configuration.
//
// This is similar to ModuleInstance alone, but specifically represents
// the module block itself rather than any one of the instances that
// module block declares.
type AbsModuleCall struct {
	Module ModuleInstance
	Call   ModuleCall
}

func (c AbsModuleCall) absMoveableSigil() {
	// AbsModuleCall is "moveable".
}

func (c AbsModuleCall) String() string {
	if len(c.Module) == 0 {
		return "module." + c.Call.Name

	}
	return fmt.Sprintf("%s.module.%s", c.Module, c.Call.Name)
}

func (c AbsModuleCall) Instance(key InstanceKey) ModuleInstance {
	ret := make(ModuleInstance, len(c.Module), len(c.Module)+1)
	copy(ret, c.Module)
	ret = append(ret, ModuleInstanceStep{
		Name:        c.Call.Name,
		InstanceKey: key,
	})
	return ret
}

func (c AbsModuleCall) Equal(other AbsModuleCall) bool {
	return c.Module.Equal(other.Module) && c.Call.Equal(other.Call)
}

type absModuleCallInstanceKey string

func (c AbsModuleCall) UniqueKey() UniqueKey {
	return absModuleCallInstanceKey(c.String())
}

func (mk absModuleCallInstanceKey) uniqueKeySigil() {}

// ModuleCallInstance is the address of one instance of a module created from
// a module call, which might create multiple instances using "count" or
// "for_each" arguments.
//
// There is no "Abs" version of ModuleCallInstance because an absolute module
// path is represented by ModuleInstance.
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

func (c ModuleCallInstance) UniqueKey() UniqueKey {
	return c // A ModuleCallInstance is its own UniqueKey
}

func (c ModuleCallInstance) uniqueKeySigil() {}

func (c ModuleCallInstance) Absolute(moduleAddr ModuleInstance) ModuleInstance {
	ret := make(ModuleInstance, len(moduleAddr), len(moduleAddr)+1)
	copy(ret, moduleAddr)
	ret = append(ret, ModuleInstanceStep{
		Name:        c.Call.Name,
		InstanceKey: c.Key,
	})
	return ret
}

// ModuleInstance returns the address of the module instance that corresponds
// to the receiving call instance when resolved in the given calling module.
// In other words, it returns the child module instance that the receving
// call instance creates.
func (c ModuleCallInstance) ModuleInstance(caller ModuleInstance) ModuleInstance {
	return caller.Child(c.Call.Name, c.Key)
}

// Output returns the absolute address of an output of the receiver identified by its
// name.
func (c ModuleCallInstance) Output(name string) ModuleCallInstanceOutput {
	return ModuleCallInstanceOutput{
		Call: c,
		Name: name,
	}
}

// ModuleCallOutput is the address of a named output and its associated
// ModuleCall, which may expand into multiple module instances
type ModuleCallOutput struct {
	referenceable
	Call ModuleCall
	Name string
}

func (m ModuleCallOutput) String() string {
	return fmt.Sprintf("%s.%s", m.Call.String(), m.Name)
}

func (m ModuleCallOutput) UniqueKey() UniqueKey {
	return m // A ModuleCallOutput is its own UniqueKey
}

func (m ModuleCallOutput) uniqueKeySigil() {}

// ModuleCallInstanceOutput is the address of a particular named output produced by
// an instance of a module call.
type ModuleCallInstanceOutput struct {
	referenceable
	Call ModuleCallInstance
	Name string
}

// ModuleCallOutput returns the referenceable ModuleCallOutput for this
// particular instance.
func (co ModuleCallInstanceOutput) ModuleCallOutput() ModuleCallOutput {
	return ModuleCallOutput{
		Call: co.Call.Call,
		Name: co.Name,
	}
}

func (co ModuleCallInstanceOutput) String() string {
	return fmt.Sprintf("%s.%s", co.Call.String(), co.Name)
}

func (co ModuleCallInstanceOutput) UniqueKey() UniqueKey {
	return co // A ModuleCallInstanceOutput is its own UniqueKey
}

func (co ModuleCallInstanceOutput) uniqueKeySigil() {}

// AbsOutputValue returns the absolute output value address that corresponds
// to the receving module call output address, once resolved in the given
// calling module.
func (co ModuleCallInstanceOutput) AbsOutputValue(caller ModuleInstance) AbsOutputValue {
	moduleAddr := co.Call.ModuleInstance(caller)
	return moduleAddr.OutputValue(co.Name)
}
