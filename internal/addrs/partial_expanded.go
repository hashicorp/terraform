// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// PartialExpandedModule represents a set of module instances which all share
// a common known parent module instance but the remaining call instance keys
// are not yet known.
type PartialExpandedModule struct {
	// module is the full module instance address, where each step whose
	// instance key is not known yet has an instance key of WildcardKey.
	//
	// The module instance keys are known for an initial prefix of the
	// address, and the remainder of the address is not expanded yet, so once
	// a step has an unknown instance key, every step after it does too. All
	// values must be constructed with newPartialExpandedModule to enforce this.
	//
	// There can be no unknown steps in PartialExpandedModule values used as
	// part of the internals of a PartialExpandedResource, but there should
	// always be at least one in a publicly-exposed PartialExpandedModule,
	// because otherwise it would be just a degenerate ModuleInstance.
	module ModuleInstance
}

// newPartialExpandedModule returns the PartialExpandedModule for the given
// module instance address, where each step with an instance key of
// WildcardKey has not been expanded yet. None of the steps after the first
// step which has not been expanded can be expanded either, so their instance
// keys are replaced with WildcardKey.
func newPartialExpandedModule(module ModuleInstance) PartialExpandedModule {
	if len(module) == 0 {
		return PartialExpandedModule{}
	}

	ret := make(ModuleInstance, len(module))
	expanded := true
	for i, step := range module {
		expanded = expanded && step.InstanceKey != WildcardKey
		if !expanded {
			step.InstanceKey = WildcardKey
		}
		ret[i] = step
	}
	return PartialExpandedModule{module: ret}
}

// ParsePartialExpandedModule parses a module address traversal and returns a
// PartialExpandedModule representing the known and unknown parts of the
// address.
//
// It returns the parsed PartialExpandedModule, the remaining traversal steps
// that were not consumed by this function, and any diagnostics that were
// generated during parsing.
func ParsePartialExpandedModule(traversal hcl.Traversal) (PartialExpandedModule, hcl.Traversal, tfdiags.Diagnostics) {
	path, remain, diags := parseModuleInstancePrefix(traversal, wildcardInstanceKeys)
	return newPartialExpandedModule(path), remain, diags
}

// UnexpandedChild returns the address of the instances of the given module
// call within the receiver, whose instance keys are not known yet.
func (m ModuleInstance) UnexpandedChild(call ModuleCall) PartialExpandedModule {
	return newPartialExpandedModule(m.Child(call.Name, WildcardKey))
}

// PartialModule reverses the process of UnknownModuleInstance by converting a
// ModuleInstance back into a PartialExpandedModule.
//
// Each step with an instance key of WildcardKey is not yet expanded, and so
// is every step after the first of them, regardless of its instance key.
func (m ModuleInstance) PartialModule() PartialExpandedModule {
	return newPartialExpandedModule(m)
}

// UnknownModuleInstance expands the receiver to a full ModuleInstance by
// replacing the unknown instance keys with a wildcard value.
func (pem PartialExpandedModule) UnknownModuleInstance() ModuleInstance {
	// As with KnownPrefix, we expose our buffer directly but with no unused
	// capacity, so that the caller can safely construct child addresses.
	return pem.module[:len(pem.module):len(pem.module)]
}

// LevelsKnown returns the number of module path segments of the address that
// have known instance keys.
//
// This might be useful, for example, for preferring a more-specifically-known
// address over a less-specifically-known one when selecting a placeholder
// value to use to represent an object beneath an unexpanded module address.
func (pem PartialExpandedModule) LevelsKnown() int {
	for i, step := range pem.module {
		if step.InstanceKey == WildcardKey {
			return i
		}
	}
	return len(pem.module)
}

// fullyExpanded returns true if all of the instance keys of the receiver are
// known, which is only possible for the module of a PartialExpandedResource or
// PartialExpandedAction.
func (pem PartialExpandedModule) fullyExpanded() bool {
	return pem.LevelsKnown() == len(pem.module)
}

// MatchesInstance returns true if and only if the given module instance
// belongs to the recieving partially-expanded module address pattern.
func (pem PartialExpandedModule) MatchesInstance(inst ModuleInstance) bool {
	return pem.matches(inst)
}

// MatchesPartial returns true if and only if the receiver represents the same
// static module as the other given module and the receiver's known instance
// keys are a prefix of the other module's.
func (pem PartialExpandedModule) MatchesPartial(other PartialExpandedModule) bool {
	return pem.matches(other.module)
}

// matches returns true if every module instance represented by the given
// address, whose unknown instance keys are WildcardKey, is also represented by
// the receiver. Unlike containment for targeting, the module instances must be
// at the same depth, rather than any descendants of the receiver.
func (pem PartialExpandedModule) matches(other ModuleInstance) bool {
	if len(other) != len(pem.module) {
		return false
	}
	return targetShape{module: pem.module}.contains(targetShape{module: other})
}

// Module returns the unexpanded module address that this pattern originated
// from.
func (pem PartialExpandedModule) Module() Module {
	return pem.module.Module()
}

// KnownPrefix returns the longest possible ModuleInstance address made of
// known segments of this partially-expanded module instance address.
func (pem PartialExpandedModule) KnownPrefix() ModuleInstance {
	known := pem.LevelsKnown()
	if known == 0 {
		return nil
	}

	// Although we can't enforce it with the Go compiler, our convention is
	// that we never mutate address values outside of this package and so
	// we'll expose our pem.module buffer directly here and trust that the
	// caller will play nice with it. However, we do force the unused capacity
	// to zero so that the caller can safely construct child addresses, which
	// would append new steps to the end.
	return pem.module[:known:known]
}

// FirstUnexpandedCall returns the address of the first step in the module
// path whose instance keys are not yet known, discarding any subsequent
// calls beneath it.
func (pem PartialExpandedModule) FirstUnexpandedCall() AbsModuleCall {
	// NOTE: This assumes that there's always at least one step with unknown
	// instance keys because it should only be used with the public-facing
	// version of PartialExpandedModule where that contract always holds. It's
	// not safe to use this for the PartialExpandedModule value hidden in the
	// internals of PartialExpandedResource.
	return AbsModuleCall{
		Module: pem.KnownPrefix(),
		Call: ModuleCall{
			Name: pem.module[pem.LevelsKnown()].Name,
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
	unexpanded := pem.module[pem.LevelsKnown():]
	if len(unexpanded) == 0 {
		// Should never happen for any publicly-visible value of this type,
		// because we should always have at least one unexpanded call,
		// but we'll allow it anyway since we have a reasonable return value
		// for that case.
		return nil
	}

	ret := make([]ModuleCall, len(unexpanded))
	for i, step := range unexpanded {
		ret[i].Name = step.Name
	}
	return ret
}

// Child returns the address of a child of the receiver that belongs to the
// given module call.
func (pem PartialExpandedModule) Child(call ModuleCall) PartialExpandedModule {
	return newPartialExpandedModule(pem.module.Child(call.Name, WildcardKey))
}

// Resource returns the address of a resource within the receiver.
func (pem PartialExpandedModule) Resource(resource Resource) PartialExpandedResource {
	return PartialExpandedResource{
		module:   pem,
		resource: resource,
	}
}

// Action returns the address of an action within the receiver.
func (pem PartialExpandedModule) Action(action Action) PartialExpandedAction {
	return PartialExpandedAction{
		module: pem,
		action: action,
	}
}

// String returns a string representation of the pattern where the known
// prefix uses the normal module instance address syntax and the unknown
// suffix steps use a similar syntax but with "[*]" as a placeholder to
// represent instance keys that aren't yet known.
func (pem PartialExpandedModule) String() string {
	return pem.module.String()
}

func (pem PartialExpandedModule) UniqueKey() UniqueKey {
	return partialExpandedModuleKey(pem.String())
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
	// This value can actually represent a fully-expanded module if none of
	// its instance keys are unknown, in which case it's only the resource
	// itself that's unexpanded, which would make this equivalent to an
	// AbsResource.
	//
	// We mustn't directly expose this value in the public API because
	// external callers must never see a PartialExpandedModule that is
	// actually fully-expanded; that should be a ModuleInstance instead.
	module   PartialExpandedModule
	resource Resource
}

// ParsePartialExpandedResource parses a resource address traversal and returns
// a PartialExpandedResource representing the known and unknown parts of the
// address.
func ParsePartialExpandedResource(traversal hcl.Traversal) (PartialExpandedResource, hcl.Traversal, tfdiags.Diagnostics) {
	pem, remain, diags := ParsePartialExpandedModule(traversal)
	if len(remain) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address",
			Detail:   "Resource address must be a module address followed by a resource address.",
			Subject:  traversal.SourceRange().Ptr(),
		})
		return PartialExpandedResource{}, nil, diags
	}

	// A PartialExpandedResource represents all of the instances of the
	// resource, so any instance key is discarded.
	resource, _, remain, moreDiags := parseResourceUnderModule(remain, wildcardInstanceKeys, true)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return PartialExpandedResource{}, nil, diags
	}

	return PartialExpandedResource{
		module:   pem,
		resource: resource,
	}, remain, diags
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
		module:   newPartialExpandedModule(m),
		resource: resource,
	}
}

// UnexpandedResource returns the receiver reinterpreted as a
// [PartialExpandedResource], which is an alternative form we use in situations
// where we might also need to mix in resources belonging to not-yet-fully-known
// module instance addresses.
func (r AbsResource) UnexpandedResource() PartialExpandedResource {
	return r.Module.UnexpandedResource(r.Resource)
}

// PartialResource reverses UnknownResourceInstance by converting the
// AbsResourceInstance back into a PartialExpandedResource.
func (r AbsResourceInstance) PartialResource() PartialExpandedResource {
	return PartialExpandedResource{
		module:   r.Module.PartialModule(),
		resource: r.Resource.Resource,
	}
}

// UnknownResourceInstance returns an [AbsResourceInstance] that represents the
// same resource as the receiver but with all instance keys replaced with a
// wildcard value.
func (per PartialExpandedResource) UnknownResourceInstance() AbsResourceInstance {
	return AbsResourceInstance{
		Module:   per.module.UnknownModuleInstance(),
		Resource: per.resource.Instance(WildcardKey),
	}
}

// MatchesInstance returns true if and only if the given resource instance
// belongs to the recieving partially-expanded resource address pattern.
func (per PartialExpandedResource) MatchesInstance(inst AbsResourceInstance) bool {
	return per.matches(inst)
}

// MatchesResource returns true if and only if the given resource belongs to
// the recieving partially-expanded resource address pattern.
func (per PartialExpandedResource) MatchesResource(inst AbsResource) bool {
	return per.matches(inst)
}

// MatchesPartial returns true if the underlying partial module address matches
// the given partial module address and the resource type and name match the
// receiver's resource type and name.
func (per PartialExpandedResource) MatchesPartial(other PartialExpandedResource) bool {
	return per.targetShape().contains(other.targetShape())
}

// matches returns true if every instance selected by the given address is
// represented by the receiver.
func (per PartialExpandedResource) matches(addr Targetable) bool {
	other, ok := shapeOf(addr)
	return ok && per.targetShape().contains(other)
}

// AbsResource returns the single [AbsResource] that this address represents
// if this pattern is specific enough to match only a single resource, or
// the zero value of AbsResource if not.
//
// The second return value is true if and only if the returned address is valid.
func (per PartialExpandedResource) AbsResource() (AbsResource, bool) {
	module, ok := per.ModuleInstance()
	if !ok {
		return AbsResource{}, false
	}

	return AbsResource{
		Module:   module,
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
	if !per.module.fullyExpanded() {
		return nil, false
	}
	return per.module.KnownPrefix(), true
}

// PartialExpandedModule returns a [PartialExpandedModule] address describing
// the partially-unknown module instance address that the resource belongs to,
// but only if the module instance address is not fully known.
//
// The second return value is false if the module instance address is actually
// fully expanded, in which case the first return value is invalid. Use
// [PartialExpandedResource.ModuleInstance] instead in that case.
func (per PartialExpandedResource) PartialExpandedModule() (PartialExpandedModule, bool) {
	if per.module.fullyExpanded() {
		return PartialExpandedModule{}, false
	}
	return per.module, true
}

// IsTargetedBy returns true if and only if the given targetable address might
// target the resource instances that could exist if the receiver were fully
// expanded.
func (per PartialExpandedResource) IsTargetedBy(addr Targetable) bool {
	target, ok := shapeOf(addr)
	if !ok {
		return false
	}
	return target.couldContain(per.targetShape())
}

// targetShape returns the shape of the resource instances the receiver could
// represent, using WildcardKey for each instance key which is not yet known.
func (per PartialExpandedResource) targetShape() targetShape {
	return targetShape{
		module:   per.module.module,
		kind:     resourceTargetShape,
		resource: per.resource,
		key:      WildcardKey,
	}
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

var _ UniqueKey = partialExpandedResourceKey("")

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

// PartialExpandedAction represents a partially-expanded action address.
// See PartialExpandedResource for more information.
type PartialExpandedAction struct {
	module PartialExpandedModule
	action Action
}

func (per PartialExpandedAction) AbsAction() (AbsAction, bool) {
	module, ok := per.ModuleInstance()
	if !ok {
		return AbsAction{}, false
	}

	return AbsAction{
		Module: module,
		Action: per.action,
	}, true
}

// ConfigAction returns the unexpanded action address that this
// partially-expanded action address originates from.
func (per PartialExpandedAction) ConfigAction() ConfigAction {
	return ConfigAction{
		Module: per.module.Module(),
		Action: per.action,
	}
}

func (per PartialExpandedAction) ModuleInstance() (ModuleInstance, bool) {
	if !per.module.fullyExpanded() {
		return nil, false
	}
	return per.module.KnownPrefix(), true
}

func (m ModuleInstance) UnexpandedAction(action Action) PartialExpandedAction {
	return PartialExpandedAction{
		module: newPartialExpandedModule(m),
		action: action,
	}
}

// UnknownActionInstance returns an [AbsActionInstance] that represents the
// same action as the receiver but with all instance keys replaced with a
// wildcard value.
func (per PartialExpandedAction) UnknownActionInstance() AbsActionInstance {
	return AbsActionInstance{
		Module: per.module.UnknownModuleInstance(),
		Action: per.action.Instance(WildcardKey),
	}
}

func (pea PartialExpandedAction) String() string {
	moduleAddr := pea.module.String()
	if len(moduleAddr) != 0 {
		return moduleAddr + "." + pea.action.String() + "[*]"
	}
	return pea.action.String() + "[*]"
}

func (pea PartialExpandedAction) Equal(other PartialExpandedAction) bool {
	return pea.module.module.Equal(other.module.module) && pea.action.Equal(other.action)
}

func (pea PartialExpandedAction) UniqueKey() UniqueKey {
	// If this address is equivalent to an AbsAction address then we'll
	// return its instance key here so that function Equivalent will consider
	// the two as equivalent.
	if ar, ok := pea.AbsAction(); ok {
		return ar.UniqueKey()
	}
	// For not-fully-expanded module paths we'll use a distinct address type
	// since there is no other address type equivalent to those.
	return partialExpandedActionKey(pea.String())
}

type partialExpandedActionKey string

func (partialExpandedActionKey) uniqueKeySigil() {}

// PartialExpandedModule returns a [PartialExpandedModule] address describing
// the partially-unknown module instance address that the action belongs to,
// but only if the module instance address is not fully known.
//
// The second return value is false if the module instance address is actually
// fully expanded, in which case the first return value is invalid. Use
// [PartialExpandedAction.ModuleInstance] instead in that case.
func (per PartialExpandedAction) PartialExpandedModule() (PartialExpandedModule, bool) {
	if per.module.fullyExpanded() {
		return PartialExpandedModule{}, false
	}
	return per.module, true
}

// MatchesAction returns true if and only if the given action belongs to
// the recieving partially-expanded action address pattern.
func (per PartialExpandedAction) MatchesAction(inst AbsAction) bool {
	other, ok := shapeOf(inst)
	return ok && per.targetShape().contains(other)
}

// targetShape returns the shape of the action instances the receiver could
// represent, using WildcardKey for each instance key which is not yet known.
func (per PartialExpandedAction) targetShape() targetShape {
	return targetShape{
		module: per.module.module,
		kind:   actionTargetShape,
		action: per.action,
		key:    WildcardKey,
	}
}
