// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"strings"
)

// Action is an address for an action block within configuration, which
// contains potentially-multiple action instances if that configuration
// block uses "count" or "for_each".
type Action struct {
	referenceable
	Type string
	Name string
}

func (a Action) String() string {
	return fmt.Sprintf("action.%s.%s", a.Type, a.Name)
}

func (a Action) Equal(o Action) bool {
	return a.Name == o.Name && a.Type == o.Type
}

func (a Action) Less(o Action) bool {
	switch {
	case a.Type != o.Type:
		return a.Type < o.Type

	case a.Name != o.Name:
		return a.Name < o.Name

	default:
		return false
	}
}

func (a Action) UniqueKey() UniqueKey {
	return a // An Action is its own UniqueKey
}

func (a Action) uniqueKeySigil() {}

// Instance produces the address for a specific instance of the receiver
// that is identified by the given key.
func (a Action) Instance(key InstanceKey) ActionInstance {
	return ActionInstance{
		Action: a,
		Key:    key,
	}
}

// Absolute returns an AbsAction from the receiver and the given module
// instance address.
func (a Action) Absolute(module ModuleInstance) AbsAction {
	return AbsAction{
		Module: module,
		Action: a,
	}
}

// InModule returns a ConfigAction from the receiver and the given module
// address.
func (a Action) InModule(module Module) ConfigAction {
	return ConfigAction{
		Module: module,
		Action: a,
	}
}

// ImpliedProvider returns the implied provider type name, for e.g. the "aws" in
// "aws_instance"
func (a Action) ImpliedProvider() string {
	typeName := a.Type
	if under := strings.Index(typeName, "_"); under != -1 {
		typeName = typeName[:under]
	}

	return typeName
}

// ActionInstance is an address for a specific instance of an action.
// When an action is defined in configuration with "count" or "for_each" it
// produces zero or more instances, which can be addressed using this type.
type ActionInstance struct {
	referenceable
	Action Action
	Key    InstanceKey
}

func (a ActionInstance) ContainingAction() Action {
	return a.Action
}

func (a ActionInstance) String() string {
	if a.Key == NoKey {
		return a.Action.String()
	}
	return a.Action.String() + a.Key.String()
}

func (a ActionInstance) Equal(o ActionInstance) bool {
	return a.Key == o.Key && a.Action.Equal(o.Action)
}

func (a ActionInstance) Less(o ActionInstance) bool {
	if !a.Action.Equal(o.Action) {
		return a.Action.Less(o.Action)
	}

	if a.Key != o.Key {
		return InstanceKeyLess(a.Key, o.Key)
	}

	return false
}

func (a ActionInstance) UniqueKey() UniqueKey {
	return a // An ActionInstance is its own UniqueKey
}

func (a ActionInstance) uniqueKeySigil() {}

// Absolute returns an AbsActionInstance from the receiver and the given module
// instance address.
func (a ActionInstance) Absolute(module ModuleInstance) AbsActionInstance {
	return AbsActionInstance{
		Module: module,
		Action: a,
	}
}

// AbsAction is an absolute address for an action under a given module path.
type AbsAction struct {
	Module ModuleInstance
	Action Action
}

// Action returns the address of a particular action within the receiver.
func (m ModuleInstance) Action(typeName string, name string) AbsAction {
	return AbsAction{
		Module: m,
		Action: Action{
			Type: typeName,
			Name: name,
		},
	}
}

// Instance produces the address for a specific instance of the receiver that is
// identified by the given key.
func (a AbsAction) Instance(key InstanceKey) AbsActionInstance {
	return AbsActionInstance{
		Module: a.Module,
		Action: a.Action.Instance(key),
	}
}

// Config returns the unexpanded ConfigAction for this AbsAction.
func (a AbsAction) Config() ConfigAction {
	return ConfigAction{
		Module: a.Module.Module(),
		Action: a.Action,
	}
}

func (a AbsAction) String() string {
	if len(a.Module) == 0 {
		return a.Action.String()
	}
	return fmt.Sprintf("%s.%s", a.Module.String(), a.Action.String())
}

// AffectedAbsAction returns the AbsAction.
func (a AbsAction) AffectedAbsAction() AbsAction {
	return a
}

func (a AbsAction) Equal(o AbsAction) bool {
	return a.Module.Equal(o.Module) && a.Action.Equal(o.Action)
}

func (a AbsAction) Less(o AbsAction) bool {
	if !a.Module.Equal(o.Module) {
		return a.Module.Less(o.Module)
	}

	if !a.Action.Equal(o.Action) {
		return a.Action.Less(o.Action)
	}

	return false
}

type absActionKey string

func (a absActionKey) uniqueKeySigil() {}

func (a AbsAction) UniqueKey() UniqueKey {
	return absActionKey(a.String())
}

// AbsActionInstance is an absolute address for an action instance under a
// given module path.
type AbsActionInstance struct {
	Module ModuleInstance
	Action ActionInstance
}

// ActionInstance returns the address of a particular action instance within the receiver.
func (m ModuleInstance) ActionInstance(typeName string, name string, key InstanceKey) AbsActionInstance {
	return AbsActionInstance{
		Module: m,
		Action: ActionInstance{
			Action: Action{
				Type: typeName,
				Name: name,
			},
			Key: key,
		},
	}
}

// ContainingAction returns the address of the action that contains the
// receiving action instance. In other words, it discards the key portion of the
// address to produce an AbsAction value.
func (a AbsActionInstance) ContainingAction() AbsAction {
	return AbsAction{
		Module: a.Module,
		Action: a.Action.ContainingAction(),
	}
}

// ConfigAction returns the address of the configuration block that declared
// this instance.
func (a AbsActionInstance) ConfigAction() ConfigAction {
	return ConfigAction{
		Module: a.Module.Module(),
		Action: a.Action.Action,
	}
}

func (a AbsActionInstance) String() string {
	if len(a.Module) == 0 {
		return a.Action.String()
	}
	return fmt.Sprintf("%s.%s", a.Module.String(), a.Action.String())
}

// AffectedAbsAction returns the AbsAction for the instance.
func (a AbsActionInstance) AffectedAbsAction() AbsAction {
	return AbsAction{
		Module: a.Module,
		Action: a.Action.Action,
	}
}

func (a AbsActionInstance) Equal(o AbsActionInstance) bool {
	return a.Module.Equal(o.Module) && a.Action.Equal(o.Action)
}

// Less returns true if the receiver should sort before the given other value
// in a sorted list of addresses.
func (a AbsActionInstance) Less(o AbsActionInstance) bool {
	if !a.Module.Equal(o.Module) {
		return a.Module.Less(o.Module)
	}

	if !a.Action.Equal(o.Action) {
		return a.Action.Less(o.Action)
	}

	return false
}

type absActionInstanceKey string

func (a AbsActionInstance) UniqueKey() UniqueKey {
	return absActionInstanceKey(a.String())
}

func (r absActionInstanceKey) uniqueKeySigil() {}

// ConfigAction is the address for an action within the configuration.
type ConfigAction struct {
	Module Module
	Action Action
}

// Action returns the address of a particular action within the module.
func (m Module) Action(typeName string, name string) ConfigAction {
	return ConfigAction{
		Module: m,
		Action: Action{
			Type: typeName,
			Name: name,
		},
	}
}

// Absolute produces the address for the receiver within a specific module instance.
func (a ConfigAction) Absolute(module ModuleInstance) AbsAction {
	return AbsAction{
		Module: module,
		Action: a.Action,
	}
}

func (a ConfigAction) String() string {
	if len(a.Module) == 0 {
		return a.Action.String()
	}
	return fmt.Sprintf("%s.%s", a.Module.String(), a.Action.String())
}

func (a ConfigAction) Equal(o ConfigAction) bool {
	return a.Module.Equal(o.Module) && a.Action.Equal(o.Action)
}

func (a ConfigAction) UniqueKey() UniqueKey {
	return configActionKey(a.String())
}

type configActionKey string

func (k configActionKey) uniqueKeySigil() {}
