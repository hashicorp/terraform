// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
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
	targetable
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

// TargetContains implements Targetable
func (a AbsAction) TargetContains(other Targetable) bool {
	switch to := other.(type) {
	case AbsAction:
		return a.Equal(to)
	case AbsActionInstance:
		return a.Equal(to.ContainingAction())
	default:
		return false
	}
}

// AddrType implements Targetable
func (a AbsAction) AddrType() TargetableAddrType {
	return ActionAddrType
}

func (a AbsAction) String() string {
	if len(a.Module) == 0 {
		return a.Action.String()
	}
	return fmt.Sprintf("%s.%s", a.Module.String(), a.Action.String())
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
	targetable
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

// TargetContains implements Targetable
func (a AbsActionInstance) TargetContains(other Targetable) bool {
	switch to := other.(type) {
	case AbsAction:
		return to.Equal(a.ContainingAction()) && a.Action.Key == NoKey
	case AbsActionInstance:
		return to.Equal(a)
	default:
		return false
	}
}

// AddrType implements Targetable
func (a AbsActionInstance) AddrType() TargetableAddrType {
	return ActionInstanceAddrType
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

func (a absActionInstanceKey) uniqueKeySigil() {}

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

// ParseAbsActionInstanceStr is a helper wrapper around
// ParseAbsActionInstance that takes a string and parses it with the HCL
// native syntax traversal parser before interpreting it.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address may be incomplete.
//
// Since this function has no context about the source of the given string,
// any returned diagnostics will not have meaningful source location
// information.
func ParseAbsActionInstanceStr(str string) (AbsActionInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsActionInstance{}, diags
	}

	addr, addrDiags := ParseAbsActionInstance(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

// ParseAbsActionInstance attempts to interpret the given traversal as an
// absolute action instance address, using the same syntax as expected by
// ParseTarget.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the AbsResource value is invalid and
// must not be used.
func ParseAbsActionInstance(traversal hcl.Traversal) (AbsActionInstance, tfdiags.Diagnostics) {
	moduleAddr, remain, diags := parseModuleInstancePrefix(traversal, false)
	if diags.HasErrors() {
		return AbsActionInstance{}, diags
	}

	if remain.IsRelative() {
		// (relative means that there's either nothing left or what's next isn't an identifier)
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action address",
			Detail:   "Module path must be followed by an action instance address.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}

	if remain.RootName() != "action" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action address must start with \"action.\".",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}
	remain = remain[1:]

	if len(remain) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Action specification must include an action type and name.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}

	var actionType, name string
	switch tt := remain[0].(type) {
	case hcl.TraverseRoot:
		actionType = tt.Name
	case hcl.TraverseAttr:
		actionType = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action address",
			Detail:   "An action name is required.",
			Subject:  remain[0].SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}

	switch tt := remain[1].(type) {
	case hcl.TraverseAttr:
		name = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action name is required.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}

	remain = remain[2:]
	switch len(remain) {
	case 0:
		return moduleAddr.ActionInstance(actionType, name, NoKey), diags
	case 1:
		switch tt := remain[0].(type) {
		case hcl.TraverseIndex:
			key, err := ParseInstanceKey(tt.Key)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid address",
					Detail:   fmt.Sprintf("Invalid resource instance key: %s.", err),
					Subject:  remain[0].SourceRange().Ptr(),
				})
				return AbsActionInstance{}, diags
			}
			return moduleAddr.ActionInstance(actionType, name, key), diags
		case hcl.TraverseSplat:
			// Not yet supported!
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Action instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			return AbsActionInstance{}, diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address",
				Detail:   "Action instance key must be given in square brackets.",
				Subject:  remain[0].SourceRange().Ptr(),
			})
			return AbsActionInstance{}, diags
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Unexpected extra operators after address.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return AbsActionInstance{}, diags
	}
}

// ParseAbsAction attempts to interpret the given traversal as an absolute
// action address, using the same syntax as expected by ParseTarget.
//
// If no error diagnostics are returned, the returned target includes the
// address that was extracted and the source range it was extracted from.
//
// If error diagnostics are returned then the AbsAction value is invalid and
// must not be used.
func ParseAbsAction(traversal hcl.Traversal) (AbsAction, tfdiags.Diagnostics) {
	addr, diags := ParseTargetAction(traversal)
	if diags.HasErrors() {
		return AbsAction{}, diags
	}

	switch tt := addr.Subject.(type) {

	case AbsAction:
		return tt, diags

	case AbsActionInstance: // Catch likely user error with specialized message
		// Assume that the last element of the traversal must be the index,
		// since that's required for a valid resource instance address.
		indexStep := traversal[len(traversal)-1]
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action address is required. This instance key identifies a specific action instance, which is not expected here.",
			Subject:  indexStep.SourceRange().Ptr(),
		})
		return AbsAction{}, diags

	case ModuleInstance: // Catch likely user error with specialized message
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action address is required here. The module path must be followed by an action specification.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsAction{}, diags

	default: // Generic message for other address types
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "An action address is required here.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return AbsAction{}, diags

	}
}

// ParseAbsActionStr is a helper wrapper around ParseAbsAction that takes a
// string and parses it with the HCL native syntax traversal parser before
// interpreting it.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address may be incomplete.
//
// Since this function has no context about the source of the given string,
// any returned diagnostics will not have meaningful source location
// information.
func ParseAbsActionStr(str string) (AbsAction, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsAction{}, diags
	}

	addr, addrDiags := ParseAbsAction(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}
