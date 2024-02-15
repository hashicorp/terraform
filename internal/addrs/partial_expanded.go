// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"strings"
)

// PartialExpandedModule represents a set of module instances which all share
// a common known parent module instance but the remaining call instance keys
// are not yet known.
type PartialExpandedModule struct {
	// expandedPrefix is the initial part of the module address whose expansion
	// is already complete and so has exact instance keys.
	expandedPrefix ModuleInstance

	// unexpandedSuffix is the remainder of the module address whose instance
	// keys are not known yet. This is a slight abuse of type [Module] because
	// it's representing a relative path from expandedPrefix rather than a
	// path from the root module as usual, so this value must never be exposed
	// in the public API of this package.
	//
	// This can be zero-length in PartialExpandedModule values used as part
	// of the internals of a PartialExpandedResource, but should never be
	// zero-length in a publicly-exposed PartialExpandedModule because that
	// would make this just a degenerate ModuleInstance.
	unexpandedSuffix Module
}

func (m ModuleInstance) UnexpandedChild(call ModuleCall) PartialExpandedModule {
	return PartialExpandedModule{
		expandedPrefix:   m,
		unexpandedSuffix: Module{call.Name},
	}
}

// LevelsKnown returns the number of module path segments of the address that
// have known instance keys.
//
// This might be useful, for example, for preferring a more-specifically-known
// address over a less-specifically-known one when selecting a placeholder
// value to use to represent an object beneath an unexpanded module address.
func (pem PartialExpandedModule) LevelsKnown() int {
	return len(pem.expandedPrefix)
}

// MatchesInstance returns true if and only if the given module instance
// belongs to the recieving partially-expanded module address pattern.
func (pem PartialExpandedModule) MatchesInstance(inst ModuleInstance) bool {
	// Total length must always match.
	if len(inst) != (len(pem.expandedPrefix) + len(pem.unexpandedSuffix)) {
		return false
	}

	// The known prefix must match exactly.
	givenExpandedPrefix := inst[:len(pem.expandedPrefix)]
	if !givenExpandedPrefix.Equal(pem.expandedPrefix) {
		return false
	}

	// The known suffix must match the call names, even though we don't yet
	// know the specific instance keys.
	givenExpandedSuffix := inst[len(pem.expandedPrefix):]
	for i := range pem.unexpandedSuffix {
		if pem.unexpandedSuffix[i] != givenExpandedSuffix[i].Name {
			return false
		}
	}

	// If we passed all the filters above then it's a match.
	return true
}

// MatchesPartial returns true if and only if the receiver represents the same
// static module as the other given module and the receiver's known instance
// keys are a prefix of the other module's.
func (pem PartialExpandedModule) MatchesPartial(other PartialExpandedModule) bool {
	// The two addresses must represent the same static module, regardless
	// of the instance keys of those modules.
	if !pem.Module().Equal(other.Module()) {
		return false
	}

	if len(pem.expandedPrefix) > len(other.expandedPrefix) {
		return false
	}

	thisPrefix := pem.expandedPrefix
	otherPrefix := other.expandedPrefix[:len(pem.expandedPrefix)]
	return thisPrefix.Equal(otherPrefix)
}

// Module returns the unexpanded module address that this pattern originated
// from.
func (pem PartialExpandedModule) Module() Module {
	ret := pem.expandedPrefix.Module()
	return append(ret, pem.unexpandedSuffix...)
}

// KnownPrefix returns the longest possible ModuleInstance address made of
// known segments of this partially-expanded module instance address.
func (pem PartialExpandedModule) KnownPrefix() ModuleInstance {
	if len(pem.expandedPrefix) == 0 {
		return nil
	}

	// Although we can't enforce it with the Go compiler, our convention is
	// that we never mutate address values outside of this package and so
	// we'll expose our pem.expandedPrefix buffer directly here and trust that
	// the caller will play nice with it. However, we do force the unused
	// capacity to zero so that the caller can safely construct child addresses,
	// which would append new steps to the end.
	return pem.expandedPrefix[:len(pem.expandedPrefix):len(pem.expandedPrefix)]
}

// FirstUnexpandedCall returns the address of the first step in the module
// path whose instance keys are not yet known, discarding any subsequent
// calls beneath it.
func (pem PartialExpandedModule) FirstUnexpandedCall() AbsModuleCall {
	// NOTE: This assumes that there's always at least one element in
	// unexpandedSuffix because it should only be used with the public-facing
	// version of PartialExpandedModule where that contract always holds. It's
	// not safe to use this for the PartialExpandedModule value hidden in the
	// internals of PartialExpandedResource.
	return AbsModuleCall{
		Module: pem.KnownPrefix(),
		Call: ModuleCall{
			Name: pem.unexpandedSuffix[0],
		},
	}
}

// UnexpandedSuffix returns the local addresses of all of the calls whose
// instances are not yet expanded, in the module tree traversal order.
//
// Method KnownPrefix concatenated with UnexpandedSuffix (assuming that were
// actually possible) represents the whole module path that the
// PartialExpandedModule encapsulates.
func (pem PartialExpandedModule) UnexpandedSuffix() []ModuleCall {
	if len(pem.unexpandedSuffix) == 0 {
		// Should never happen for any publicly-visible value of this type,
		// because we should always have at least one unexpanded call,
		// but we'll allow it anyway since we have a reasonable return value
		// for that case.
		return nil
	}

	// A []ModuleCall is the only representation of a non-rooted chain of
	// module calls that we're allowed to export in our public API, and so
	// we'll transform our not-quite-allowed unrooted "Module" value in that
	// form externally.
	ret := make([]ModuleCall, len(pem.unexpandedSuffix))
	for i, name := range pem.unexpandedSuffix {
		ret[i].Name = name
	}
	return ret
}

// Child returns the address of a child of the receiver that belongs to the
// given module call.
func (pem PartialExpandedModule) Child(call ModuleCall) PartialExpandedModule {
	return PartialExpandedModule{
		expandedPrefix:   pem.expandedPrefix,
		unexpandedSuffix: append(pem.unexpandedSuffix, call.Name),
	}
}

// Resource returns the address of a resource within the receiver.
func (pem PartialExpandedModule) Resource(resource Resource) PartialExpandedResource {
	return PartialExpandedResource{
		module:   pem,
		resource: resource,
	}
}

// String returns a string representation of the pattern where the known
// prefix uses the normal module instance address syntax and the unknown
// suffix steps use a similar syntax but with "[*]" as a placeholder to
// represent instance keys that aren't yet known.
func (pem PartialExpandedModule) String() string {
	var buf strings.Builder
	if len(pem.expandedPrefix) != 0 {
		buf.WriteString(pem.expandedPrefix.String())
	}
	for i, callName := range pem.unexpandedSuffix {
		if i > 0 || len(pem.expandedPrefix) != 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("module.")
		buf.WriteString(callName)
		buf.WriteString("[*]")
	}
	return buf.String()
}

func (per PartialExpandedModule) UniqueKey() UniqueKey {
	return partialExpandedModuleKey(per.String())
}

type partialExpandedModuleKey string

var _ UniqueKey = partialExpandedModuleKey("")

func (partialExpandedModuleKey) uniqueKeySigil() {}

// PartialExpandedResource represents a set of resource instances which all share
// a common known parent module instance but the remaining call instance keys
// are not yet known and the resource's own instance keys are not yet known.
//
// A PartialExpandedResource with a fully-known module instance address is
// semantically interchangable with an [AbsResource], which is useful when we
// need to represent an assortment of variously-unknown resource instance
// addresses, but [AbsResource] is preferable in situations where the module
// instance address is _always_ known and it's only the resource instance
// key that is not represented.
type PartialExpandedResource struct {
	// module is the partially-expanded module instance address that this
	// resource belongs to.
	//
	// This value can actually represent a fully-expanded module if its
	// unexpandedSuffix field is zero-length, in which case it's only the
	// resource itself that's unexpanded, which would make this equivalent
	// to an AbsResource.
	//
	// We mustn't directly expose this value in the public API because
	// external callers must never see a PartialExpandedModule that is
	// actually fully-expanded; that should be a ModuleInstance instead.
	module   PartialExpandedModule
	resource Resource
}

// UnexpandedResource returns the address of a child resource expressed as a
// [PartialExpandedResource].
//
// The result always has a fully-qualified module instance address and is
// therefore semantically equivalent to an [AbsResource], so this variannt
// should be used only in contexts where we might also be storing resources
// belonging to not-fully-expanded modules and need to use the same static
// address type for all of them.
func (m ModuleInstance) UnexpandedResource(resource Resource) PartialExpandedResource {
	return PartialExpandedResource{
		module: PartialExpandedModule{
			expandedPrefix: m,
		},
		resource: resource,
	}
}

// UnexpandedResource returns the receiver reinterpreted as a
// [PartialExpandedResource], which is an alternative form we use in situations
// where we might also need to mix in resources belonging to not-yet-fully-known
// module instance addresses.
func (r AbsResource) UnexpandedResource() PartialExpandedResource {
	return PartialExpandedResource{
		module: PartialExpandedModule{
			expandedPrefix: r.Module,
		},
		resource: r.Resource,
	}
}

// MatchesInstance returns true if and only if the given resource instance
// belongs to the recieving partially-expanded resource address pattern.
func (per PartialExpandedResource) MatchesInstance(inst AbsResourceInstance) bool {
	if !per.module.MatchesInstance(inst.Module) {
		return false
	}
	return inst.Resource.Resource.Equal(per.resource)
}

// MatchesResource returns true if and only if the given resource belongs to
// the recieving partially-expanded resource address pattern.
func (per PartialExpandedResource) MatchesResource(inst AbsResource) bool {
	if !per.module.MatchesInstance(inst.Module) {
		return false
	}
	return inst.Resource.Equal(per.resource)
}

// AbsResource returns the single [AbsResource] that this address represents
// if this pattern is specific enough to match only a single resource, or
// the zero value of AbsResource if not.
//
// The second return value is true if and only if the returned address is valid.
func (per PartialExpandedResource) AbsResource() (AbsResource, bool) {
	if len(per.module.unexpandedSuffix) != 0 {
		return AbsResource{}, false
	}

	return AbsResource{
		Module:   per.module.expandedPrefix,
		Resource: per.resource,
	}, true
}

// ConfigResource returns the unexpanded resource address that this
// partially-expanded resource address originates from.
func (per PartialExpandedResource) ConfigResource() ConfigResource {
	return ConfigResource{
		Module:   per.module.Module(),
		Resource: per.resource,
	}
}

// Resource returns just the leaf resource address that this partially-expanded
// resource address uses, discarding the containing module instance information
// altogether.
func (per PartialExpandedResource) Resource() Resource {
	return per.resource
}

// KnownModuleInstancePrefix returns the longest possible ModuleInstance address
// made of known segments of the module instances that this set of resource
// instances all belong to.
//
// If the whole module instance address is known and only the resource
// instances are not then this returns the full prefix, which will be the same
// as the module from a successful return value from
// [PartialExpandedResource.AbsResource].
func (per PartialExpandedResource) KnownModuleInstancePrefix() ModuleInstance {
	return per.module.KnownPrefix()
}

// ModuleInstance returns the fully-qualified [ModuleInstance] that this
// partial-expanded resource belongs to, but only if its module instance
// address is fully known.
//
// The second return value is false if the module instance address is not
// fully expanded, in which case the first return value is invalid. Use
// [PartialExpandedResource.PartialExpandedModule] instead in that case.
func (per PartialExpandedResource) ModuleInstance() (ModuleInstance, bool) {
	if len(per.module.unexpandedSuffix) != 0 {
		return nil, false
	}
	return per.module.expandedPrefix, true
}

// PartialExpandedModule returns a [PartialExpandedModule] address describing
// the partially-unknown module instance address that the resource belongs to,
// but only if the module instance address is not fully known.
//
// The second return value is false if the module instance address is actually
// fully expanded, in which case the first return value is invalid. Use
// [PartialExpandedResource.ModuleInstance] instead in that case.
func (per PartialExpandedResource) PartialExpandedModule() (PartialExpandedModule, bool) {
	if len(per.module.unexpandedSuffix) == 0 {
		return PartialExpandedModule{}, false
	}
	return per.module, true
}

// String returns a string representation of the pattern which uses the special
// placeholder "[*]" to represent positions where instance keys are not yet
// known.
func (per PartialExpandedResource) String() string {
	moduleAddr := per.module.String()
	if len(moduleAddr) != 0 {
		return moduleAddr + "." + per.resource.String() + "[*]"
	}
	return per.resource.String() + "[*]"
}

func (per PartialExpandedResource) UniqueKey() UniqueKey {
	// If this address is equivalent to an AbsResource address then we'll
	// return its instance key here so that function Equivalent will consider
	// the two as equivalent.
	if ar, ok := per.AbsResource(); ok {
		return ar.UniqueKey()
	}
	// For not-fully-expanded module paths we'll use a distinct address type
	// since there is no other address type equivalent to those.
	return partialExpandedResourceKey(per.String())
}

type partialExpandedResourceKey string

var _ UniqueKey = partialExpandedModuleKey("")

func (partialExpandedResourceKey) uniqueKeySigil() {}

// InPartialExpandedModule is a generic type used for all address types that
// represent objects that exist inside module instances but do not have any
// expansion capability of their own beyond just the containing module
// expansion.
//
// Although not enforced by the type system, this type should be used only for
// address types T that are combined with a ModuleInstance value in a type
// whose name starts with "Abs". For example, [LocalValue] is a reasonable T
// because [AbsLocalValue] represents a local value inside a particular module
// instance. InPartialExpandedModule[LocalValue] is therefore like an
// [AbsLocalValue] whose module path isn't fully known yet.
//
// This type is here primarily just to have implementations of [UniqueKeyer]
// so we can store partially-evaluated objects from unexpanded modules in
// collections for later reference downstream.
type InPartialExpandedModule[T interface {
	UniqueKeyer
	fmt.Stringer
}] struct {
	Module PartialExpandedModule
	Local  T
}

// ObjectInPartialExpandedModule is a constructor for [InPartialExpandedModule]
// that's here primarily just to benefit from function type parameter inference
// to avoid manually writing out type T when constructing such a value.
func ObjectInPartialExpandedModule[T interface {
	UniqueKeyer
	fmt.Stringer
}](module PartialExpandedModule, local T) InPartialExpandedModule[T] {
	return InPartialExpandedModule[T]{
		Module: module,
		Local:  local,
	}
}

var _ UniqueKeyer = InPartialExpandedModule[LocalValue]{}

// ModuleLevelsKnown returns the number of module path segments of the address
// that have known instance keys.
//
// This might be useful, for example, for preferring a more-specifically-known
// address over a less-specifically-known one when selecting a placeholder
// value to use to represent an object beneath an unexpanded module address.
func (in InPartialExpandedModule[T]) ModuleLevelsKnown() int {
	return in.Module.LevelsKnown()
}

// String returns a string representation of the pattern which uses the special
// placeholder "[*]" to represent positions where module instance keys are not
// yet known.
func (in InPartialExpandedModule[T]) String() string {
	moduleAddr := in.Module.String()
	if len(moduleAddr) != 0 {
		return moduleAddr + "." + in.Local.String()
	}
	return in.Local.String()
}

func (in InPartialExpandedModule[T]) UniqueKey() UniqueKey {
	return inPartialExpandedModuleUniqueKey{
		moduleKey: in.Module.UniqueKey(),
		localKey:  in.Local.UniqueKey(),
	}
}

type inPartialExpandedModuleUniqueKey struct {
	moduleKey UniqueKey
	localKey  UniqueKey
}

func (inPartialExpandedModuleUniqueKey) uniqueKeySigil() {}
