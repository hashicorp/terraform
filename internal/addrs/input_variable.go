// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
)

// InputVariable is the address of an input variable.
type InputVariable struct {
	referenceable
	Name string
}

func (v InputVariable) String() string {
	return "var." + v.Name
}

func (v InputVariable) UniqueKey() UniqueKey {
	return v // A InputVariable is its own UniqueKey
}

func (v InputVariable) uniqueKeySigil() {}

// Absolute converts the receiver into an absolute address within the given
// module instance.
func (v InputVariable) Absolute(m ModuleInstance) AbsInputVariableInstance {
	return AbsInputVariableInstance{
		Module:   m,
		Variable: v,
	}
}

func (v InputVariable) InModule(module Module) ConfigInputVariable {
	return ConfigInputVariable{
		Module:   module,
		Variable: v,
	}
}

// AbsInputVariableInstance is the address of an input variable within a
// particular module instance.
type AbsInputVariableInstance struct {
	Module   ModuleInstance
	Variable InputVariable
}

var _ Checkable = AbsInputVariableInstance{}

// InputVariable returns the absolute address of the input variable of the
// given name inside the receiving module instance.
func (m ModuleInstance) InputVariable(name string) AbsInputVariableInstance {
	return AbsInputVariableInstance{
		Module: m,
		Variable: InputVariable{
			Name: name,
		},
	}
}

func (v AbsInputVariableInstance) String() string {
	if len(v.Module) == 0 {
		return v.Variable.String()
	}

	return fmt.Sprintf("%s.%s", v.Module.String(), v.Variable.String())
}

func (v AbsInputVariableInstance) UniqueKey() UniqueKey {
	return absInputVariableInstanceUniqueKey(v.String())
}

func (v AbsInputVariableInstance) checkableSigil() {}

func (v AbsInputVariableInstance) CheckRule(typ CheckRuleType, i int) CheckRule {
	return CheckRule{
		Container: v,
		Type:      typ,
		Index:     i,
	}
}

func (v AbsInputVariableInstance) ConfigCheckable() ConfigCheckable {
	return ConfigInputVariable{
		Module:   v.Module.Module(),
		Variable: v.Variable,
	}
}

func (v AbsInputVariableInstance) CheckableKind() CheckableKind {
	return CheckableInputVariable
}

type ConfigInputVariable struct {
	Module   Module
	Variable InputVariable
}

var _ ConfigCheckable = ConfigInputVariable{}

func (v ConfigInputVariable) UniqueKey() UniqueKey {
	return configInputVariableUniqueKey(v.String())
}

func (v ConfigInputVariable) configCheckableSigil() {}

func (v ConfigInputVariable) CheckableKind() CheckableKind {
	return CheckableInputVariable
}

func (v ConfigInputVariable) String() string {
	if len(v.Module) == 0 {
		return v.Variable.String()
	}

	return fmt.Sprintf("%s.%s", v.Module.String(), v.Variable.String())
}

type configInputVariableUniqueKey string

func (k configInputVariableUniqueKey) uniqueKeySigil() {}

type absInputVariableInstanceUniqueKey string

func (k absInputVariableInstanceUniqueKey) uniqueKeySigil() {}
