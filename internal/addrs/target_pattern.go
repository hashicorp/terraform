// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"strings"
)

// TargetPattern is a Targetable address which selects every module instance,
// resource instance, or action instance which matches a pattern.
//
// Target addresses given by users are parsed as patterns, using ParseTarget or
// ParseTargetAction. Each step of the module path, and the instance key of the
// resource or action, is either a specific instance key, or a wildcard which
// selects every instance. A wildcard can be written explicitly as [*], or
// implicitly by omitting the instance key, which retains the meaning of the
// original target syntax where an address without an instance key refers to
// all of the instances.
//
// Because a pattern can select multiple instances, it is distinct from the
// concrete address types, where an omitted instance key refers only to an
// instance without a key.
type TargetPattern struct {
	targetable

	shape targetShape

	// explicitModule records which module steps used an explicit [*]
	// wildcard, and explicitKey records the same for the resource or action
	// instance key, so that String can reproduce the original syntax.
	explicitModule []bool
	explicitKey    bool
}

var _ Targetable = TargetPattern{}

// newTargetPattern converts an address returned from parsing a traversal
// pattern into a TargetPattern. Instance keys of WildcardKey were written as
// [*], while those of NoKey were omitted, and both select every instance.
func newTargetPattern(module ModuleInstance, kind targetShapeKind, resource Resource, action Action, key InstanceKey) TargetPattern {
	p := TargetPattern{
		shape: targetShape{
			module:   make(ModuleInstance, len(module)),
			kind:     kind,
			resource: resource,
			action:   action,
		},
		explicitModule: make([]bool, len(module)),
	}

	for i, step := range module {
		p.explicitModule[i] = step.InstanceKey == WildcardKey
		if step.InstanceKey == NoKey {
			step.InstanceKey = WildcardKey
		}
		p.shape.module[i] = step
	}

	if kind != moduleTargetShape {
		p.explicitKey = key == WildcardKey
		if key == NoKey {
			key = WildcardKey
		}
		p.shape.key = key
	}

	return p
}

// Contains implements Targetable by returning true if every address
// selected by the given other address is also selected by the receiver.
func (p TargetPattern) Contains(other Targetable) bool {
	return targetContains(p, other)
}

// ConfigAddr returns the address of the configuration object which declares
// all of the instances selected by the pattern, which is either a Module, a
// ConfigResource, or a ConfigAction.
func (p TargetPattern) ConfigAddr() Targetable {
	mod := p.shape.module.Module()
	switch p.shape.kind {
	case resourceTargetShape:
		return ConfigResource{Module: mod, Resource: p.shape.resource}
	case actionTargetShape:
		return ConfigAction{Module: mod, Action: p.shape.action}
	default:
		return mod
	}
}

// InstanceKey returns the instance key of the resource or action selected by
// the pattern, which is WildcardKey when every instance is selected. Patterns
// selecting a module always return NoKey.
func (p TargetPattern) InstanceKey() InstanceKey {
	return p.shape.key
}

// String returns the pattern using the same syntax it was parsed from, which
// can be parsed again with ParseTarget or ParseTargetAction.
func (p TargetPattern) String() string {
	return p.format(false)
}

// Equal returns true if both patterns select the same addresses, regardless
// of whether their wildcards are written explicitly.
func (p TargetPattern) Equal(other TargetPattern) bool {
	return p.UniqueKey() == other.UniqueKey()
}

// UniqueKey implements UniqueKeyer. Patterns which differ only in whether
// their wildcards are written explicitly select the same addresses, and so
// have the same key.
func (p TargetPattern) UniqueKey() UniqueKey {
	return targetPatternKey(p.format(true))
}

type targetPatternKey string

func (k targetPatternKey) uniqueKeySigil() {}

// format renders the pattern, writing all wildcards explicitly if
// allExplicit is set.
func (p TargetPattern) format(allExplicit bool) string {
	var buf strings.Builder
	formatKey := func(key InstanceKey, explicit bool) {
		switch key {
		case NoKey:
		case WildcardKey:
			if explicit || allExplicit {
				buf.WriteString(key.String())
			}
		default:
			buf.WriteString(key.String())
		}
	}

	for i, step := range p.shape.module {
		if i > 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("module.")
		buf.WriteString(step.Name)
		formatKey(step.InstanceKey, p.explicitModule[i])
	}

	var object string
	switch p.shape.kind {
	case resourceTargetShape:
		object = p.shape.resource.String()
	case actionTargetShape:
		object = p.shape.action.String()
	default:
		return buf.String()
	}
	if buf.Len() > 0 {
		buf.WriteByte('.')
	}
	buf.WriteString(object)
	formatKey(p.shape.key, p.explicitKey)
	return buf.String()
}

type targetShapeKind int

const (
	moduleTargetShape targetShapeKind = iota
	resourceTargetShape
	actionTargetShape
)

// targetShape is a common representation of every Targetable address, which
// allows Contains to use the same rules for all address types.
//
// Any step of the module path, or the instance key, can be WildcardKey to
// select every instance. Configuration addresses select every instance of
// every module, and so all of their steps are wildcards. Addresses of a whole
// resource or action select every instance, so use WildcardKey for the key.
type targetShape struct {
	module   ModuleInstance
	kind     targetShapeKind
	resource Resource
	action   Action
	// key is only used for resources and actions
	key InstanceKey
}

// targetContains implements Contains for every Targetable address type,
// so they all follow the same rules.
func targetContains(container, other Targetable) bool {
	c, ok := shapeOf(container)
	if !ok {
		return false
	}
	o, ok := shapeOf(other)
	return ok && c.contains(o)
}

// CouldContain returns true if the container could contain any of the
// addresses which other represents, once all of their instance keys are known.
//
// Unlike Contains, any wildcard in other is treated as an instance key which
// is not known yet, rather than as every instance. A configuration address is
// treated the same way, because none of its instance keys are known, so this
// decides whether a target could select any instance of an object which has
// not been expanded yet. For an address without any wildcards, this is the
// same as Contains.
func CouldContain(container, other Targetable) bool {
	c, ok := shapeOf(container)
	if !ok {
		return false
	}
	o, ok := shapeOf(other)
	return ok && c.couldContain(o)
}

func shapeOf(addr Targetable) (targetShape, bool) {
	switch addr := addr.(type) {
	case TargetPattern:
		return addr.shape, true
	case Module:
		return targetShape{module: allModuleInstances(addr)}, true
	case ModuleInstance:
		return targetShape{module: addr}, true
	case ConfigResource:
		return targetShape{module: allModuleInstances(addr.Module), kind: resourceTargetShape, resource: addr.Resource, key: WildcardKey}, true
	case AbsResource:
		return targetShape{module: addr.Module, kind: resourceTargetShape, resource: addr.Resource, key: WildcardKey}, true
	case AbsResourceInstance:
		return targetShape{module: addr.Module, kind: resourceTargetShape, resource: addr.Resource.Resource, key: addr.Resource.Key}, true
	case ConfigAction:
		return targetShape{module: allModuleInstances(addr.Module), kind: actionTargetShape, action: addr.Action, key: WildcardKey}, true
	case AbsAction:
		return targetShape{module: addr.Module, kind: actionTargetShape, action: addr.Action, key: WildcardKey}, true
	case AbsActionInstance:
		return targetShape{module: addr.Module, kind: actionTargetShape, action: addr.Action.Action, key: addr.Action.Key}, true
	default:
		return targetShape{}, false
	}
}

// allModuleInstances returns the module instance address which selects every
// instance of the given module.
func allModuleInstances(m Module) ModuleInstance {
	ret := make(ModuleInstance, len(m))
	for i, name := range m {
		ret[i] = ModuleInstanceStep{Name: name, InstanceKey: WildcardKey}
	}
	return ret
}

// contains returns true if every address selected by other is also selected
// by the receiver. A module contains everything within it, while resources
// and actions only contain their own instances.
func (s targetShape) contains(other targetShape) bool {
	return s.compare(other, func(key, otherKey InstanceKey) bool {
		return key == WildcardKey || key == otherKey
	})
}

// couldContain is like contains, but treats any wildcards in other as
// instance keys which are not yet known, and so returns true if any of the
// addresses which other could eventually represent might be contained by the
// receiver.
func (s targetShape) couldContain(other targetShape) bool {
	return s.compare(other, func(key, otherKey InstanceKey) bool {
		return key == WildcardKey || otherKey == WildcardKey || key == otherKey
	})
}

func (s targetShape) compare(other targetShape, keysMatch func(key, otherKey InstanceKey) bool) bool {
	if len(other.module) < len(s.module) {
		return false
	}
	for i, step := range s.module {
		otherStep := other.module[i]
		if step.Name != otherStep.Name || !keysMatch(step.InstanceKey, otherStep.InstanceKey) {
			return false
		}
	}

	switch s.kind {
	case moduleTargetShape:
		return true
	case resourceTargetShape:
		return other.kind == resourceTargetShape &&
			len(other.module) == len(s.module) &&
			s.resource.Equal(other.resource) &&
			keysMatch(s.key, other.key)
	case actionTargetShape:
		return other.kind == actionTargetShape &&
			len(other.module) == len(s.module) &&
			s.action.Equal(other.action) &&
			keysMatch(s.key, other.key)
	default:
		return false
	}
}
