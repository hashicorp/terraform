package addrs

import (
	"fmt"
)

// OutputValue is the address of an output value, in the context of the module
// that is defining it.
//
// This is related to but separate from ModuleCallOutput, which represents
// a module output from the perspective of its parent module. Since output
// values cannot be represented from the module where they are defined,
// OutputValue is not Referenceable, while ModuleCallOutput is.
type OutputValue struct {
	Name string
}

func (v OutputValue) String() string {
	return "output." + v.Name
}

// Absolute converts the receiver into an absolute address within the given
// module instance.
func (v OutputValue) Absolute(m ModuleInstance) AbsOutputValue {
	return AbsOutputValue{
		Module:      m,
		OutputValue: v,
	}
}

// AbsOutputValue is the absolute address of an output value within a module instance.
//
// This represents an output globally within the namespace of a particular
// configuration. It is related to but separate from ModuleCallOutput, which
// represents a module output from the perspective of its parent module.
type AbsOutputValue struct {
	Module      ModuleInstance
	OutputValue OutputValue
}

// OutputValue returns the absolute address of an output value of the given
// name within the receiving module instance.
func (m ModuleInstance) OutputValue(name string) AbsOutputValue {
	return AbsOutputValue{
		Module: m,
		OutputValue: OutputValue{
			Name: name,
		},
	}
}

func (v AbsOutputValue) String() string {
	if v.Module.IsRoot() {
		return v.OutputValue.String()
	}
	return fmt.Sprintf("%s.%s", v.Module.String(), v.OutputValue.String())
}

// ModuleCallOutput converts an AbsModuleOutput into a ModuleCallOutput,
// returning also the module instance that the ModuleCallOutput is relative
// to.
//
// The root module does not have a call, and so this method cannot be used
// with outputs in the root module, and will panic in that case.
func (v AbsOutputValue) ModuleCallOutput() (ModuleInstance, AbsModuleCallOutput) {
	if v.Module.IsRoot() {
		panic("ReferenceFromCall used with root module output")
	}

	caller, call := v.Module.CallInstance()
	return caller, AbsModuleCallOutput{
		Call: call,
		Name: v.OutputValue.Name,
	}
}
