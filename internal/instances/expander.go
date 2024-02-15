// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instances

import (
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Expander instances serve as a coordination point for gathering object
// repetition values (count and for_each in configuration) and then later
// making use of them to fully enumerate all of the instances of an object.
//
// The two repeatable object types in Terraform are modules and resources.
// Because resources belong to modules and modules can nest inside other
// modules, module expansion in particular has a recursive effect that can
// cause deep objects to expand exponentially. Expander assumes that all
// instances of a module have the same static objects inside, and that they
// differ only in the repetition count for some of those objects.
//
// Expander is a synchronized object whose methods can be safely called
// from concurrent threads of execution. However, it does expect a certain
// sequence of operations which is normally obtained by the caller traversing
// a dependency graph: each object must have its repetition mode set exactly
// once, and this must be done before any calls that depend on the repetition
// mode. In other words, the count or for_each expression value for a module
// must be provided before any object nested directly or indirectly inside
// that module can be expanded. If this ordering is violated, the methods
// will panic to enforce internal consistency.
//
// The Expand* methods of Expander only work directly with modules and with
// resources. Addresses for other objects that nest within modules but
// do not themselves support repetition can be obtained by calling ExpandModule
// with the containing module path and then producing one absolute instance
// address per module instance address returned.
type Expander struct {
	mu   sync.RWMutex
	exps *expanderModule
}

// NewExpander initializes and returns a new Expander, empty and ready to use.
func NewExpander() *Expander {
	return &Expander{
		exps: newExpanderModule(),
	}
}

// SetModuleSingle records that the given module call inside the given parent
// module does not use any repetition arguments and is therefore a singleton.
func (e *Expander) SetModuleSingle(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall) {
	e.setModuleExpansion(parentAddr, callAddr, expansionSingleVal)
}

// SetModuleCount records that the given module call inside the given parent
// module instance uses the "count" repetition argument, with the given value.
func (e *Expander) SetModuleCount(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall, count int) {
	e.setModuleExpansion(parentAddr, callAddr, expansionCount(count))
}

// SetModuleCountUnknown records that the given module call inside the given
// parent module instance uses the "count" repetition argument but its value
// is not yet known.
func (e *Expander) SetModuleCountUnknown(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall) {
	e.setModuleExpansion(parentAddr, callAddr, expansionDeferredIntKey)
}

// SetModuleForEach records that the given module call inside the given parent
// module instance uses the "for_each" repetition argument, with the given
// map value.
//
// In the configuration language the for_each argument can also accept a set.
// It's the caller's responsibility to convert that into an identity map before
// calling this method.
func (e *Expander) SetModuleForEach(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall, mapping map[string]cty.Value) {
	e.setModuleExpansion(parentAddr, callAddr, expansionForEach(mapping))
}

// SetModuleForEachUnknown records that the given module call inside the given
// parent module instance uses the "for_each" repetition argument, but its
// map keys are not yet known.
func (e *Expander) SetModuleForEachUnknown(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall) {
	e.setModuleExpansion(parentAddr, callAddr, expansionDeferredStringKey)
}

// SetResourceSingle records that the given resource inside the given module
// does not use any repetition arguments and is therefore a singleton.
func (e *Expander) SetResourceSingle(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource) {
	e.setResourceExpansion(moduleAddr, resourceAddr, expansionSingleVal)
}

// SetResourceCount records that the given resource inside the given module
// uses the "count" repetition argument, with the given value.
func (e *Expander) SetResourceCount(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource, count int) {
	e.setResourceExpansion(moduleAddr, resourceAddr, expansionCount(count))
}

// SetResourceCountUnknown records that the given resource inside the given
// module uses the "count" repetition argument but its value isn't yet known.
func (e *Expander) SetResourceCountUnknown(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource) {
	e.setResourceExpansion(moduleAddr, resourceAddr, expansionDeferredIntKey)
}

// SetResourceForEach records that the given resource inside the given module
// uses the "for_each" repetition argument, with the given map value.
//
// In the configuration language the for_each argument can also accept a set.
// It's the caller's responsibility to convert that into an identity map before
// calling this method.
func (e *Expander) SetResourceForEach(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource, mapping map[string]cty.Value) {
	e.setResourceExpansion(moduleAddr, resourceAddr, expansionForEach(mapping))
}

// SetResourceForEachUnknown records that the given resource inside the given
// module uses the "for_each" repetition argument, but the map keys aren't
// known yet.
func (e *Expander) SetResourceForEachUnknown(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource) {
	e.setResourceExpansion(moduleAddr, resourceAddr, expansionDeferredStringKey)
}

// ExpandModule finds the exhaustive set of module instances resulting from
// the expansion of the given module and all of its ancestor modules.
//
// If any involved module calls have an as-yet-unknown set of instance keys
// then the result includes only the known instance addresses, if any.
//
// All of the modules on the path to the identified module must already have
// had their expansion registered using one of the SetModule* methods before
// calling, or this method will panic.
func (e *Expander) ExpandModule(addr addrs.Module) []addrs.ModuleInstance {
	return e.expandModule(addr, false)
}

// ExpandAbsModuleCall is similar to [Expander.ExpandModule] except that it
// filters the result to include only the instances that belong to the
// given module call instance, and therefore returns just instance keys
// since the rest of the module address is implied by the given argument.
//
// For example, passing an address representing module.a["foo"].module.b
// would include only instances under module.a["foo"], and disregard instances
// under other dynamic paths like module.a["bar"].
//
// If the requested module call has an unknown expansion (e.g. because it
// had an unknown value for count or for_each) then the second result is
// false and the other results are meaningless. If the second return value is
// true, then the set of module instances is complete, and all of the instances
// have instance keys matching the returned keytype.
//
// The instances are returned in the typical sort order for the returned
// key type: integer keys are sorted numerically, and string keys are sorted
// lexically.
func (e *Expander) ExpandAbsModuleCall(addr addrs.AbsModuleCall) (keyType addrs.InstanceKeyType, insts []addrs.InstanceKey, known bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	expParent, ok := e.findModule(addr.Module)
	if !ok {
		// This module call lives under an unknown-expansion prefix, so we
		// cannot answer this question.
		return addrs.NoKeyType, nil, false
	}

	expCall, ok := expParent.moduleCalls[addr.Call]
	if !ok {
		// This indicates a bug, since we should've calculated the expansions
		// (even if unknown) before any caller asks for the results.
		panic(fmt.Sprintf("no expansion has been registered for %s", addr.String()))
	}
	keyType, instKeys, deferred := expCall.instanceKeys()
	if deferred {
		return addrs.NoKeyType, nil, false
	}
	return keyType, instKeys, true
}

// expandModule allows skipping unexpanded module addresses by setting skipUnregistered to true.
// This is used by instances.Set, which is only concerned with the expanded
// instances, and should not panic when looking up unknown addresses.
func (e *Expander) expandModule(addr addrs.Module, skipUnregistered bool) []addrs.ModuleInstance {
	if len(addr) == 0 {
		// Root module is always a singleton.
		return singletonRootModule
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// We're going to be dynamically growing ModuleInstance addresses, so
	// we'll preallocate some space to do it so that for typical shallow
	// module trees we won't need to reallocate this.
	// (moduleInstances does plenty of allocations itself, so the benefit of
	// pre-allocating this is marginal but it's not hard to do.)
	parentAddr := make(addrs.ModuleInstance, 0, 4)
	ret := e.exps.moduleInstances(addr, parentAddr, skipUnregistered)
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Less(ret[j])
	})
	return ret
}

// UnknownModuleInstances finds a set of patterns that collectively cover
// all of the possible module instance addresses that could appear for the
// given module once all of the intermediate module expansions are fully known.
//
// This imprecisely describes what's omitted from the [Expander.ExpandModule]
// result whenever there's an as-yet-unknown call expansion somewhere in the
// module path.
//
// Note that an [addrs.PartialExpandedModule] value is effectively an infinite
// set of [addrs.ModuleInstance] values itself, so the result could be
// considered as the union of all of those sets but we return it as a set of
// sets because the inner sets are of infinite size while the outer set is
// finite.
func (e *Expander) UnknownModuleInstances(addr addrs.Module) addrs.Set[addrs.PartialExpandedModule] {
	if len(addr) == 0 {
		// The root module is always "expanded" because it's always a singleton,
		// so we have nothing to return in that case.
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	ret := addrs.MakeSet[addrs.PartialExpandedModule]()
	parentAddr := make(addrs.ModuleInstance, 0, 4)
	e.exps.partialExpandedModuleInstances(addr, parentAddr, ret)
	return ret
}

// GetDeepestExistingModuleInstance is a funny specialized function for
// determining how many steps we can traverse through the given module instance
// address before encountering an undeclared instance of a declared module.
//
// The result is the longest prefix of the given address which steps only
// through module instances that exist.
//
// All of the modules on the given path must already have had their
// expansion registered using one of the SetModule* methods before calling,
// or this method will panic.
func (e *Expander) GetDeepestExistingModuleInstance(given addrs.ModuleInstance) addrs.ModuleInstance {
	exps := e.exps // start with the root module expansions
	for i := 0; i < len(given); i++ {
		step := given[i]
		callName := step.Name
		if _, ok := exps.moduleCalls[addrs.ModuleCall{Name: callName}]; !ok {
			// This is a bug in the caller, because it should always register
			// expansions for an object and all of its ancestors before requesting
			// expansion of it.
			panic(fmt.Sprintf("no expansion has been registered for %s", given[:i].Child(callName, addrs.NoKey)))
		}

		var ok bool
		exps, ok = exps.childInstances[step]
		if !ok {
			// We've found a non-existing instance, so we're done.
			return given[:i]
		}
	}

	// If we complete the loop above without returning early then the entire
	// given address refers to a declared module instance.
	return given
}

// ExpandModuleResource finds the exhaustive set of resource instances resulting from
// the expansion of the given resource and all of its containing modules.
//
// If any involved module calls or resources have an as-yet-unknown set of
// instance keys then the result includes only the known instance addresses,
// if any.
//
// All of the modules on the path to the identified resource and the resource
// itself must already have had their expansion registered using one of the
// SetModule*/SetResource* methods before calling, or this method will panic.
func (e *Expander) ExpandModuleResource(moduleAddr addrs.Module, resourceAddr addrs.Resource) []addrs.AbsResourceInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// We're going to be dynamically growing ModuleInstance addresses, so
	// we'll preallocate some space to do it so that for typical shallow
	// module trees we won't need to reallocate this.
	// (moduleInstances does plenty of allocations itself, so the benefit of
	// pre-allocating this is marginal but it's not hard to do.)
	moduleInstanceAddr := make(addrs.ModuleInstance, 0, 4)
	ret := e.exps.moduleResourceInstances(moduleAddr, resourceAddr, moduleInstanceAddr)
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Less(ret[j])
	})
	return ret
}

// ExpandResource finds the set of resource instances resulting from
// the expansion of the given resource within its module instance.
//
// All of the modules on the path to the identified resource and the resource
// itself must already have had their expansion registered using one of the
// SetModule*/SetResource* methods before calling, or this method will panic.
//
// ExpandModuleResource returns all instances of a resource across all
// instances of its containing module, whereas this ExpandResource function
// is more specific and only expands within a single module instance. If
// any of the module instances selected in the module path of the given address
// aren't valid for that module's expansion then ExpandResource returns an
// empty result, reflecting that a non-existing module instance can never
// contain any existing resource instances.
func (e *Expander) ExpandResource(resourceAddr addrs.AbsResource) []addrs.AbsResourceInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	moduleInstanceAddr := make(addrs.ModuleInstance, 0, 4)
	ret := e.exps.resourceInstances(resourceAddr.Module, resourceAddr.Resource, moduleInstanceAddr)
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Less(ret[j])
	})
	return ret
}

// UnknownResourceInstances finds a set of patterns that collectively cover
// all of the possible resource instance addresses that could appear for the
// given static resource once all of the intermediate module expansions are
// fully known.
//
// This imprecisely describes what's omitted from the [Expander.ExpandResource]
// and [Expander.ExpandModuleResource] results whenever there's an
// as-yet-unknown expansion somewhere in the module path or in the resource
// itself.
//
// Note that an [addrs.PartialExpandedResource] value is effectively an infinite
// set of [addrs.AbsResourceInstance] values itself, so the result could be
// considered as the union of all of those sets but we return it as a set of
// sets because the inner sets are of infinite size while the outer set is
// finite.
func (e *Expander) UnknownResourceInstances(resourceAddr addrs.ConfigResource) addrs.Set[addrs.PartialExpandedResource] {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ret := addrs.MakeSet[addrs.PartialExpandedResource]()
	parentModuleAddr := make(addrs.ModuleInstance, 0, 4)
	e.exps.partialExpandedResourceInstances(resourceAddr.Module, resourceAddr.Resource, parentModuleAddr, ret)
	return ret
}

// GetModuleInstanceRepetitionData returns an object describing the values
// that should be available for each.key, each.value, and count.index within
// the call block for the given module instance.
func (e *Expander) GetModuleInstanceRepetitionData(addr addrs.ModuleInstance) RepetitionData {
	if len(addr) == 0 {
		// The root module is always a singleton, so it has no repetition data.
		return RepetitionData{}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addr[:len(addr)-1])
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know what type of expansion we're doing.
		return TotallyUnknownRepetitionData
	}
	lastStep := addr[len(addr)-1]
	exp, ok := parentMod.moduleCalls[addrs.ModuleCall{Name: lastStep.Name}]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr))
	}
	return exp.repetitionData(lastStep.InstanceKey)
}

// GetModuleCallInstanceKeys determines the child instance keys for one specific
// instance of a module call.
//
// keyType describes the expected type of all keys in knownKeys, which typically
// also implies what data type would be used to describe the full set of
// instances: [addrs.IntKeyType] as a list or tuple, [addrs.StringKeyType] as
// a map or object, and [addrs.NoKeyType] as just a single value.
//
// If unknownKeys is true then there might be additional keys that we can't know
// yet because the call's expansion isn't known.
func (e *Expander) GetModuleCallInstanceKeys(addr addrs.AbsModuleCall) (keyType addrs.InstanceKeyType, knownKeys []addrs.InstanceKey, unknownKeys bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addr.Module)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know yet what kind of instance key to expect. (The caller might
		// be able to infer this itself using configuration info, though.)
		return addrs.UnknownKeyType, nil, true
	}
	exp, ok := parentMod.moduleCalls[addr.Call]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr))
	}
	return exp.instanceKeys()
}

// GetResourceInstanceRepetitionData returns an object describing the values
// that should be available for each.key, each.value, and count.index within
// the definition block for the given resource instance.
func (e *Expander) GetResourceInstanceRepetitionData(addr addrs.AbsResourceInstance) RepetitionData {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addr.Module)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know what type of expansion we're doing.
		return TotallyUnknownRepetitionData
	}
	exp, ok := parentMod.resources[addr.Resource.Resource]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr.ContainingResource()))
	}
	return exp.repetitionData(addr.Resource.Key)
}

// ResourceInstanceKeys determines the child instance keys for one specific
// instance of a resource.
//
// keyType describes the expected type of all keys in knownKeys, which typically
// also implies what data type would be used to describe the full set of
// instances: [addrs.IntKeyType] as a list or tuple, [addrs.StringKeyType] as
// a map or object, and [addrs.NoKeyType] as just a single value.
//
// If unknownKeys is true then there might be additional keys that we can't know
// yet because the call's expansion isn't known.
func (e *Expander) ResourceInstanceKeys(addr addrs.AbsResource) (keyType addrs.InstanceKeyType, knownKeys []addrs.InstanceKey, unknownKeys bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addr.Module)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know yet what kind of instance key to expect. (The caller might
		// be able to infer this itself using configuration info, though.)
		return addrs.UnknownKeyType, nil, true
	}
	exp, ok := parentMod.resources[addr.Resource]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr))
	}
	return exp.instanceKeys()
}

// AllInstances returns a set of all of the module and resource instances known
// to the expander.
//
// It generally doesn't make sense to call this until everything has already
// been fully expanded by calling the SetModule* and SetResource* functions.
// After that, the returned set is a convenient small API only for querying
// whether particular instance addresses appeared as a result of those
// expansions.
func (e *Expander) AllInstances() Set {
	return Set{e}
}

func (e *Expander) findModule(moduleInstAddr addrs.ModuleInstance) (expMod *expanderModule, known bool) {
	// We expect that all of the modules on the path to our module instance
	// should already have expansions registered.
	mod := e.exps
	for i, step := range moduleInstAddr {
		if expansionIsDeferred(mod.moduleCalls[addrs.ModuleCall{Name: step.Name}]) {
			return nil, false
		}
		next, ok := mod.childInstances[step]
		if !ok {
			// Top-down ordering of registration is part of the contract of
			// Expander, so this is always indicative of a bug in the caller.
			panic(fmt.Sprintf("no expansion has been registered for ancestor module %s", moduleInstAddr[:i+1]))
		}
		mod = next
	}
	return mod, true
}

func (e *Expander) setModuleExpansion(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall, exp expansion) {
	e.mu.Lock()
	defer e.mu.Unlock()

	mod, known := e.findModule(parentAddr)
	if !known {
		panic(fmt.Sprintf("can't register expansion for call in %s beneath unexpanded parent", parentAddr))
	}
	if _, exists := mod.moduleCalls[callAddr]; exists {
		panic(fmt.Sprintf("expansion already registered for %s", parentAddr.Child(callAddr.Name, addrs.NoKey)))
	}
	if !expansionIsDeferred(exp) {
		// We'll also pre-register the child instances so that later calls can
		// populate them as the caller traverses the configuration tree.
		_, knownKeys, _ := exp.instanceKeys()
		for _, key := range knownKeys {
			step := addrs.ModuleInstanceStep{Name: callAddr.Name, InstanceKey: key}
			mod.childInstances[step] = newExpanderModule()
		}
	}
	mod.moduleCalls[callAddr] = exp
}

func (e *Expander) setResourceExpansion(parentAddr addrs.ModuleInstance, resourceAddr addrs.Resource, exp expansion) {
	e.mu.Lock()
	defer e.mu.Unlock()

	mod, known := e.findModule(parentAddr)
	if !known {
		panic(fmt.Sprintf("can't register expansion in %s where path includes unknown expansion", parentAddr))
	}
	if _, exists := mod.resources[resourceAddr]; exists {
		panic(fmt.Sprintf("expansion already registered for %s", resourceAddr.Absolute(parentAddr)))
	}
	mod.resources[resourceAddr] = exp
}

func (e *Expander) knowsModuleInstance(want addrs.ModuleInstance) bool {
	if want.IsRoot() {
		return true // root module instance is always present
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	return e.exps.knowsModuleInstance(want)
}

func (e *Expander) knowsModuleCall(want addrs.AbsModuleCall) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.exps.knowsModuleCall(want)
}

func (e *Expander) knowsResourceInstance(want addrs.AbsResourceInstance) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.exps.knowsResourceInstance(want)
}

func (e *Expander) knowsResource(want addrs.AbsResource) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.exps.knowsResource(want)
}

type expanderModule struct {
	moduleCalls    map[addrs.ModuleCall]expansion
	resources      map[addrs.Resource]expansion
	childInstances map[addrs.ModuleInstanceStep]*expanderModule
}

func newExpanderModule() *expanderModule {
	return &expanderModule{
		moduleCalls:    make(map[addrs.ModuleCall]expansion),
		resources:      make(map[addrs.Resource]expansion),
		childInstances: make(map[addrs.ModuleInstanceStep]*expanderModule),
	}
}

var singletonRootModule = []addrs.ModuleInstance{addrs.RootModuleInstance}

// if moduleInstances is being used to lookup known instances after all
// expansions have been done, set skipUnregistered to true which allows addrs
// which may not have been seen to return with no instances rather than
// panicking.
func (m *expanderModule) moduleInstances(addr addrs.Module, parentAddr addrs.ModuleInstance, skipUnregistered bool) []addrs.ModuleInstance {
	callName := addr[0]
	exp, ok := m.moduleCalls[addrs.ModuleCall{Name: callName}]
	if !ok {
		if skipUnregistered {
			return nil
		}
		// This is a bug in the caller, because it should always register
		// expansions for an object and all of its ancestors before requesting
		// expansion of it.
		panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Child(callName, addrs.NoKey)))
	}
	if expansionIsDeferred(exp) {
		// We don't yet have enough information to determine the instance
		// addresses for this module.
		return nil
	}

	var ret []addrs.ModuleInstance

	// If there's more than one step remaining then we need to traverse deeper.
	if len(addr) > 1 {
		for step, inst := range m.childInstances {
			if step.Name != callName {
				continue
			}
			instAddr := append(parentAddr, step)
			ret = append(ret, inst.moduleInstances(addr[1:], instAddr, skipUnregistered)...)
		}
		return ret
	}

	// Otherwise, we'll use the expansion from the final step to produce
	// a sequence of addresses under this prefix.
	_, knownKeys, _ := exp.instanceKeys()
	for _, k := range knownKeys {
		// We're reusing the buffer under parentAddr as we recurse through
		// the structure, so we need to copy it here to produce a final
		// immutable slice to return.
		full := make(addrs.ModuleInstance, 0, len(parentAddr)+1)
		full = append(full, parentAddr...)
		full = full.Child(callName, k)
		ret = append(ret, full)
	}
	return ret
}

func (m *expanderModule) partialExpandedModuleInstances(addr addrs.Module, parentAddr addrs.ModuleInstance, into addrs.Set[addrs.PartialExpandedModule]) {
	callName := addr[0]
	exp, ok := m.moduleCalls[addrs.ModuleCall{Name: callName}]
	if !ok {
		// This is a bug in the caller, because it should always register
		// expansions for an object and all of its ancestors before requesting
		// expansion of it.
		panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Child(callName, addrs.NoKey)))
	}
	if expansionIsDeferred(exp) {
		// We've found a deferred expansion, so we're done searching this
		// subtree and can just treat the whole of "addr" as unexpanded
		// calls.
		retAddr := parentAddr.UnexpandedChild(addrs.ModuleCall{Name: callName})
		for _, step := range addr[1:] {
			retAddr = retAddr.Child(addrs.ModuleCall{Name: step})
		}
		into.Add(retAddr)
		return
	}

	// If this step already has everything expanded then we need to
	// search inside it to see if it has any unexpanded descendents.
	if len(addr) > 1 {
		for step, inst := range m.childInstances {
			if step.Name != callName {
				continue
			}
			instAddr := append(parentAddr, step)
			inst.partialExpandedModuleInstances(addr[1:], instAddr, into)
		}
	}
}

func (m *expanderModule) moduleResourceInstances(moduleAddr addrs.Module, resourceAddr addrs.Resource, parentAddr addrs.ModuleInstance) []addrs.AbsResourceInstance {
	if len(moduleAddr) > 0 {
		var ret []addrs.AbsResourceInstance
		// We need to traverse through the module levels first, so we can
		// then iterate resource expansions in the context of each module
		// path leading to them.
		callName := moduleAddr[0]
		if exp, ok := m.moduleCalls[addrs.ModuleCall{Name: callName}]; !ok {
			// This is a bug in the caller, because it should always register
			// expansions for an object and all of its ancestors before requesting
			// expansion of it.
			panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Child(callName, addrs.NoKey)))
		} else if expansionIsDeferred(exp) {
			// We don't yet have any known instance addresses, then.
			return nil
		}

		for step, inst := range m.childInstances {
			if step.Name != callName {
				continue
			}
			moduleInstAddr := append(parentAddr, step)
			ret = append(ret, inst.moduleResourceInstances(moduleAddr[1:], resourceAddr, moduleInstAddr)...)
		}
		return ret
	}

	return m.onlyResourceInstances(resourceAddr, parentAddr)
}

func (m *expanderModule) resourceInstances(moduleAddr addrs.ModuleInstance, resourceAddr addrs.Resource, parentAddr addrs.ModuleInstance) []addrs.AbsResourceInstance {
	if len(moduleAddr) > 0 {
		// We need to traverse through the module levels first, using only the
		// module instances for our specific resource, as the resource may not
		// yet be expanded in all module instances.
		step := moduleAddr[0]
		callName := step.Name
		if _, ok := m.moduleCalls[addrs.ModuleCall{Name: callName}]; !ok {
			// This is a bug in the caller, because it should always register
			// expansions for an object and all of its ancestors before requesting
			// expansion of it.
			panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Child(callName, addrs.NoKey)))
		}

		if inst, ok := m.childInstances[step]; ok {
			moduleInstAddr := append(parentAddr, step)
			return inst.resourceInstances(moduleAddr[1:], resourceAddr, moduleInstAddr)
		} else {
			// If we have the module _call_ registered (as we checked above)
			// but we don't have the given module _instance_ registered, that
			// suggests that the module instance key in "step" is not declared
			// by the current definition of this module call. That means the
			// module instance doesn't exist at all, and therefore it can't
			// possibly declare any resource instances either.
			//
			// For example, if we were asked about module.foo[0].aws_instance.bar
			// but module.foo doesn't currently have count set, then there is no
			// module.foo[0] at all, and therefore no aws_instance.bar
			// instances inside it.
			return nil
		}
	}
	return m.onlyResourceInstances(resourceAddr, parentAddr)
}

func (m *expanderModule) partialExpandedResourceInstances(moduleAddr addrs.Module, resourceAddr addrs.Resource, parentAddr addrs.ModuleInstance, into addrs.Set[addrs.PartialExpandedResource]) {
	// The idea here is to recursively walk along the module path until we
	// either encounter a module call whose expansion isn't known yet or we
	// run out of module steps. If we make it all the way to the end of the
	// module path without encountering anything then that just leaves the
	// resource expansion, which itself might be either known or unknown.

	switch {
	case len(moduleAddr) > 0:
		callName := moduleAddr[0]
		callAddr := addrs.ModuleCall{Name: callName}
		exp, ok := m.moduleCalls[callAddr]
		if !ok {
			// This is a bug in the caller, because it should always register
			// expansions for an object and all of its ancestors before requesting
			// expansion of it.
			panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Child(callName, addrs.NoKey)))
		}
		if expansionIsDeferred(exp) {
			// We've found a module call with an unknown expansion so this is
			// as far as we can go and the rest of the module path has
			// unknown expansion.
			retMod := parentAddr.UnexpandedChild(callAddr)
			for _, stepName := range moduleAddr[1:] {
				retMod = retMod.Child(addrs.ModuleCall{Name: stepName})
			}
			ret := retMod.Resource(resourceAddr)
			into.Add(ret)
			return
		}

		// If we get here then we can continue exploring all of the known
		// instances of this current module call.
		for step, inst := range m.childInstances {
			if step.Name != callName {
				continue
			}
			instAddr := parentAddr.Child(step.Name, step.InstanceKey)
			inst.partialExpandedResourceInstances(moduleAddr[1:], resourceAddr, instAddr, into)
		}

	default:
		// If we've run out of module address steps then the only remaining
		// question is whether the resource's own expansion is known.
		exp, ok := m.resources[resourceAddr]
		if !ok {
			// This is a bug in the caller, because it should always register
			// expansions for an object and all of its ancestors before requesting
			// expansion of it.
			panic(fmt.Sprintf("no expansion has been registered for %s", parentAddr.Resource(resourceAddr.Mode, resourceAddr.Type, resourceAddr.Name)))
		}
		if expansionIsDeferred(exp) {
			ret := parentAddr.UnexpandedResource(resourceAddr)
			into.Add(ret)
			return
		}
		// If the expansion isn't deferred then there's nothing to do here,
		// because the instances of this resource would appear in the
		// resourceInstances method results instead.
	}
}

func (m *expanderModule) onlyResourceInstances(resourceAddr addrs.Resource, parentAddr addrs.ModuleInstance) []addrs.AbsResourceInstance {
	var ret []addrs.AbsResourceInstance
	exp, ok := m.resources[resourceAddr]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", resourceAddr.Absolute(parentAddr)))
	}
	if expansionIsDeferred(exp) {
		// We don't yet have enough information to determine the instance addresses.
		return nil
	}

	_, knownKeys, _ := exp.instanceKeys()
	for _, k := range knownKeys {
		// We're reusing the buffer under parentAddr as we recurse through
		// the structure, so we need to copy it here to produce a final
		// immutable slice to return.
		moduleAddr := make(addrs.ModuleInstance, len(parentAddr))
		copy(moduleAddr, parentAddr)
		ret = append(ret, resourceAddr.Instance(k).Absolute(moduleAddr))
	}
	return ret
}

func (m *expanderModule) getModuleInstance(want addrs.ModuleInstance) *expanderModule {
	current := m
	for _, step := range want {
		next := current.childInstances[step]
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func (m *expanderModule) knowsModuleInstance(want addrs.ModuleInstance) bool {
	return m.getModuleInstance(want) != nil
}

func (m *expanderModule) knowsModuleCall(want addrs.AbsModuleCall) bool {
	modInst := m.getModuleInstance(want.Module)
	if modInst == nil {
		return false
	}
	_, ret := modInst.moduleCalls[want.Call]
	return ret
}

func (m *expanderModule) knowsResourceInstance(want addrs.AbsResourceInstance) bool {
	modInst := m.getModuleInstance(want.Module)
	if modInst == nil {
		return false
	}
	resourceExp := modInst.resources[want.Resource.Resource]
	if resourceExp == nil {
		return false
	}
	_, knownKeys, _ := resourceExp.instanceKeys()
	for _, key := range knownKeys {
		if key == want.Resource.Key {
			return true
		}
	}
	return false
}

func (m *expanderModule) knowsResource(want addrs.AbsResource) bool {
	modInst := m.getModuleInstance(want.Module)
	if modInst == nil {
		return false
	}
	_, ret := modInst.resources[want.Resource]
	return ret
}
