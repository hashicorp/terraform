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
func newTargetPattern(addr targetShape) TargetPattern {
	parsed := addr.shapeModule()
	p := TargetPattern{
		explicitModule: make([]bool, len(parsed)),
	}

	module := make(ModuleInstance, len(parsed))
	for i, step := range parsed {
		p.explicitModule[i] = step.InstanceKey == WildcardKey
		if step.InstanceKey == NoKey {
			step.InstanceKey = WildcardKey
		}
		module[i] = step
	}

	key := shapeKey(addr)
	p.explicitKey = key == WildcardKey
	if key == NoKey {
		key = WildcardKey
	}

	switch addr := addr.(type) {
	case AbsResourceInstance:
		p.shape = addr.Resource.Resource.Instance(key).Absolute(module)
	case AbsActionInstance:
		p.shape = addr.Action.Action.Instance(key).Absolute(module)
	default:
		p.shape = module
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
	switch shape := p.shape.(type) {
	case AbsResourceInstance:
		return shape.ConfigResource()
	case AbsActionInstance:
		return shape.ConfigAction()
	case ModuleInstance:
		return shape.Module()
	default:
		return RootModule
	}
}

// InstanceKey returns the instance key of the resource or action selected by
// the pattern, which is WildcardKey when every instance is selected. Patterns
// selecting a module always return NoKey.
func (p TargetPattern) InstanceKey() InstanceKey {
	return shapeKey(p.shape)
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

	if p.shape == nil {
		return ""
	}

	for i, step := range p.shape.shapeModule() {
		if i > 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("module.")
		buf.WriteString(step.Name)
		formatKey(step.InstanceKey, p.explicitModule[i])
	}

	var object string
	switch shape := p.shape.(type) {
	case AbsResourceInstance:
		object = shape.Resource.Resource.String()
	case AbsActionInstance:
		object = shape.Action.Action.String()
	default:
		return buf.String()
	}
	if buf.Len() > 0 {
		buf.WriteByte('.')
	}
	buf.WriteString(object)
	formatKey(shapeKey(p.shape), p.explicitKey)
	return buf.String()
}

// targetShape is a common representation of every Targetable address, which
// allows Contains to use the same rules for all address types. It's
// implemented only by ModuleInstance, AbsResourceInstance, and
// AbsActionInstance.
//
// Any step of the module path, or the instance key of the resource or action,
// can be WildcardKey to select every instance. Configuration addresses select
// every instance of every module, and so all of their steps are wildcards.
// Addresses of a whole resource or action select every instance, so use
// WildcardKey for its instance key.
type targetShape interface {
	Targetable

	// shapeModule returns the module path of the address. Only the address
	// types which can be a targetShape implement it.
	shapeModule() ModuleInstance
}

var (
	_ targetShape = ModuleInstance(nil)
	_ targetShape = AbsResourceInstance{}
	_ targetShape = AbsActionInstance{}
)

func (m ModuleInstance) shapeModule() ModuleInstance      { return m }
func (r AbsResourceInstance) shapeModule() ModuleInstance { return r.Module }
func (a AbsActionInstance) shapeModule() ModuleInstance   { return a.Module }

// shapeKey returns the instance key of the resource or action in the given
// shape, or NoKey for a module.
func shapeKey(s targetShape) InstanceKey {
	switch s := s.(type) {
	case AbsResourceInstance:
		return s.Resource.Key
	case AbsActionInstance:
		return s.Action.Key
	default:
		return NoKey
	}
}

// targetContains implements Contains for every Targetable address type,
// so they all follow the same rules.
func targetContains(container, other Targetable) bool {
	c, ok := shapeOf(container)
	if !ok {
		return false
	}
	o, ok := shapeOf(other)
	return ok && shapeContains(c, o)
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
	return ok && shapeCouldContain(c, o)
}

func shapeOf(addr Targetable) (targetShape, bool) {
	switch addr := addr.(type) {
	case TargetPattern:
		return addr.shape, addr.shape != nil
	case Module:
		return allModuleInstances(addr), true
	case ModuleInstance:
		return addr, true
	case ConfigResource:
		return addr.Resource.Instance(WildcardKey).Absolute(allModuleInstances(addr.Module)), true
	case AbsResource:
		return addr.Instance(WildcardKey), true
	case AbsResourceInstance:
		return addr, true
	case ConfigAction:
		return addr.Action.Instance(WildcardKey).Absolute(allModuleInstances(addr.Module)), true
	case AbsAction:
		return addr.Instance(WildcardKey), true
	case AbsActionInstance:
		return addr, true
	default:
		return nil, false
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

// shapeContains returns true if every address selected by other is also
// selected by s. A module contains everything within it, while resources and
// actions only contain their own instances.
func shapeContains(s, other targetShape) bool {
	return shapeCompare(s, other, func(key, otherKey InstanceKey) bool {
		return key == WildcardKey || key == otherKey
	})
}

// shapeCouldContain is like shapeContains, but treats any wildcards in other
// as instance keys which are not yet known, and so returns true if any of the
// addresses which other could eventually represent might be contained by s.
func shapeCouldContain(s, other targetShape) bool {
	return shapeCompare(s, other, func(key, otherKey InstanceKey) bool {
		return key == WildcardKey || otherKey == WildcardKey || key == otherKey
	})
}

func shapeCompare(s, other targetShape, keysMatch func(key, otherKey InstanceKey) bool) bool {
	module, otherModule := s.shapeModule(), other.shapeModule()
	if len(otherModule) < len(module) {
		return false
	}
	for i, step := range module {
		otherStep := otherModule[i]
		if step.Name != otherStep.Name || !keysMatch(step.InstanceKey, otherStep.InstanceKey) {
			return false
		}
	}

	switch s := s.(type) {
	case ModuleInstance:
		// A module contains everything within it.
		return true
	case AbsResourceInstance:
		o, ok := other.(AbsResourceInstance)
		return ok && len(o.Module) == len(s.Module) &&
			s.Resource.Resource.Equal(o.Resource.Resource) &&
			keysMatch(s.Resource.Key, o.Resource.Key)
	case AbsActionInstance:
		o, ok := other.(AbsActionInstance)
		return ok && len(o.Module) == len(s.Module) &&
			s.Action.Action.Equal(o.Action.Action) &&
			keysMatch(s.Action.Key, o.Action.Key)
	default:
		return false
	}
}
