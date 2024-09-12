// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"fmt"
	"maps"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

// State is the top-level type of a Terraform state.
//
// A state should be mutated only via its accessor methods, to ensure that
// invariants are preserved.
//
// Access to State and the nested values within it is not concurrency-safe,
// so when accessing a State object concurrently it is the caller's
// responsibility to ensure that only one write is in progress at a time
// and that reads only occur when no write is in progress. The most common
// way to achieve this is to wrap the State in a SyncState and use the
// higher-level atomic operations supported by that type.
type State struct {
	// Modules contains the state for each module. The keys in this map are
	// an implementation detail and must not be used by outside callers.
	Modules map[string]*Module

	// RootOutputValues contains the state for each non-ephemeral output value
	// defined in the root module.
	//
	// Output values in other modules don't persist anywhere between runs,
	// so Terraform Core tracks those only internally and does not expose
	// them in any artifacts that survive between runs.
	RootOutputValues map[string]*OutputValue

	// EphemeralRootOutputValues contains the state for each ephemeral output
	// value defined in the root module.
	//
	// Ephemeral outputs are treated separately from non-ephemeral outputs, to
	// ensure that their values are never written to the state file.
	EphemeralRootOutputValues map[string]*OutputValue

	// CheckResults contains a snapshot of the statuses of checks at the
	// end of the most recent update to the state. Callers might compare
	// checks between runs to see if e.g. a previously-failing check has
	// been fixed since the last run, or similar.
	//
	// CheckResults can be nil to indicate that there are no check results
	// from the previous run at all, which is subtly different than the
	// previous run having affirmatively recorded that there are no checks
	// to run. For example, if this object was created from a state snapshot
	// created by a version of Terraform that didn't yet support checks
	// then this field will be nil.
	CheckResults *CheckResults
}

// NewState constructs a minimal empty state, containing an empty root module.
func NewState() *State {
	modules := map[string]*Module{}
	modules[addrs.RootModuleInstance.String()] = NewModule(addrs.RootModuleInstance)
	return &State{
		Modules:                   modules,
		RootOutputValues:          make(map[string]*OutputValue),
		EphemeralRootOutputValues: make(map[string]*OutputValue),
	}
}

// BuildState is a helper -- primarily intended for tests -- to build a state
// using imperative code against the StateSync type while still acting as
// an expression of type *State to assign into a containing struct.
func BuildState(cb func(*SyncState)) *State {
	s := NewState()
	cb(s.SyncWrapper())
	return s
}

// Empty returns true if there are no resources or populated output values
// in the receiver. In other words, if this state could be safely replaced
// with the return value of NewState and be functionally equivalent.
func (s *State) Empty() bool {
	if s == nil {
		return true
	}
	if len(s.RootOutputValues) != 0 || len(s.EphemeralRootOutputValues) != 0 {
		return false
	}
	for _, ms := range s.Modules {
		if !ms.empty() {
			return false
		}
	}
	return true
}

// Module returns the state for the module with the given address, or nil if
// the requested module is not tracked in the state.
func (s *State) Module(addr addrs.ModuleInstance) *Module {
	if s == nil {
		panic("State.Module on nil *State")
	}
	return s.Modules[addr.String()]
}

// ModuleInstances returns the set of Module states that matches the given path.
func (s *State) ModuleInstances(addr addrs.Module) []*Module {
	var ms []*Module
	for _, m := range s.Modules {
		if m.Addr.Module().Equal(addr) {
			ms = append(ms, m)
		}
	}
	return ms
}

// RemoveModule removes the module with the given address from the state,
// unless it is the root module. The root module cannot be deleted, and so
// this method will panic if that is attempted.
//
// Removing a module implicitly discards all of the resources, outputs and
// local values within it, and so this should usually be done only for empty
// modules. For callers accessing the state through a SyncState wrapper, modules
// are automatically pruned if they are empty after one of their contained
// elements is removed.
func (s *State) RemoveModule(addr addrs.ModuleInstance) {
	if addr.IsRoot() {
		panic("attempted to remove root module")
	}

	delete(s.Modules, addr.String())
}

// RootModule is a convenient alias for Module(addrs.RootModuleInstance).
func (s *State) RootModule() *Module {
	if s == nil {
		panic("RootModule called on nil State")
	}
	return s.Modules[addrs.RootModuleInstance.String()]
}

// EnsureModule returns the state for the module with the given address,
// creating and adding a new one if necessary.
//
// Since this might modify the state to add a new instance, it is considered
// to be a write operation.
func (s *State) EnsureModule(addr addrs.ModuleInstance) *Module {
	ms := s.Module(addr)
	if ms == nil {
		ms = NewModule(addr)
		s.Modules[addr.String()] = ms
	}
	return ms
}

// HasManagedResourceInstanceObjects returns true if there is at least one
// resource instance object (current or deposed) associated with a managed
// resource in the receiving state.
//
// A true result would suggest that just discarding this state without first
// destroying these objects could leave "dangling" objects in remote systems,
// no longer tracked by any Terraform state.
func (s *State) HasManagedResourceInstanceObjects() bool {
	if s == nil {
		return false
	}
	for _, ms := range s.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			for _, is := range rs.Instances {
				if is.Current != nil || len(is.Deposed) != 0 {
					return true
				}
			}
		}
	}
	return false
}

// Resource returns the state for the resource with the given address, or nil
// if no such resource is tracked in the state.
func (s *State) Resource(addr addrs.AbsResource) *Resource {
	ms := s.Module(addr.Module)
	if ms == nil {
		return nil
	}
	return ms.Resource(addr.Resource)
}

// Resources returns the set of resources that match the given configuration path.
func (s *State) Resources(addr addrs.ConfigResource) []*Resource {
	var ret []*Resource
	for _, m := range s.ModuleInstances(addr.Module) {
		r := m.Resource(addr.Resource)
		if r != nil {
			ret = append(ret, r)
		}
	}
	return ret
}

// AllResourceInstanceObjectAddrs returns a set of addresses for all of
// the leaf resource instance objects of any mode that are tracked in this
// state.
//
// If you only care about objects belonging to managed resources, use
// [State.AllManagedResourceInstanceObjectAddrs] instead.
func (s *State) AllResourceInstanceObjectAddrs() addrs.Set[addrs.AbsResourceInstanceObject] {
	return s.allResourceInstanceObjectAddrs(func(addr addrs.AbsResourceInstanceObject) bool {
		return true // we filter nothing
	})
}

// AllManagedResourceInstanceObjectAddrs returns a set of addresses for all of
// the leaf resource instance objects associated with managed resources that
// are tracked in this state.
//
// This result is the set of objects that would be effectively "forgotten"
// (like "terraform state rm") if this state were totally discarded, such as
// by deleting a workspace. This function is intended only for reporting
// context in error messages, such as when we reject deleting a "non-empty"
// workspace as detected by s.HasManagedResourceInstanceObjects.
func (s *State) AllManagedResourceInstanceObjectAddrs() addrs.Set[addrs.AbsResourceInstanceObject] {
	return s.allResourceInstanceObjectAddrs(func(addr addrs.AbsResourceInstanceObject) bool {
		return addr.ResourceInstance.Resource.Resource.Mode == addrs.ManagedResourceMode
	})
}

func (s *State) allResourceInstanceObjectAddrs(keepAddr func(addr addrs.AbsResourceInstanceObject) bool) addrs.Set[addrs.AbsResourceInstanceObject] {
	if s == nil {
		return nil
	}

	ret := addrs.MakeSet[addrs.AbsResourceInstanceObject]()
	for _, ms := range s.Modules {
		for _, rs := range ms.Resources {
			for instKey, is := range rs.Instances {
				instAddr := rs.Addr.Instance(instKey)
				if is.Current != nil {
					objAddr := addrs.AbsResourceInstanceObject{
						ResourceInstance: instAddr,
						DeposedKey:       addrs.NotDeposed,
					}
					if keepAddr(objAddr) {
						ret.Add(objAddr)
					}
				}
				for deposedKey := range is.Deposed {
					objAddr := addrs.AbsResourceInstanceObject{
						ResourceInstance: instAddr,
						DeposedKey:       deposedKey,
					}
					if keepAddr(objAddr) {
						ret.Add(objAddr)
					}
				}
			}
		}
	}

	return ret
}

// ResourceInstance returns the state for the resource instance with the given
// address, or nil if no such resource is tracked in the state.
func (s *State) ResourceInstance(addr addrs.AbsResourceInstance) *ResourceInstance {
	if s == nil {
		panic("State.ResourceInstance on nil *State")
	}
	ms := s.Module(addr.Module)
	if ms == nil {
		return nil
	}
	return ms.ResourceInstance(addr.Resource)
}

// ResourceInstance returns the (encoded) state for the resource instance object
// with the given address, or nil if no such object is tracked in the state.
func (s *State) ResourceInstanceObjectSrc(addr addrs.AbsResourceInstanceObject) *ResourceInstanceObjectSrc {
	if s == nil {
		panic("State.ResourceInstanceObjectSrc on nil *State")
	}
	rs := s.ResourceInstance(addr.ResourceInstance)
	if rs == nil {
		return nil
	}
	if addr.DeposedKey != addrs.NotDeposed {
		return rs.Deposed[addr.DeposedKey]
	}
	return rs.Current
}

// OutputValue returns the state for the output value with the given address,
// or nil if no such output value is tracked in the state.
//
// Only root module output values are tracked in the state, so this always
// returns nil for output values in any other module.
func (s *State) OutputValue(addr addrs.AbsOutputValue) *OutputValue {
	if !addr.Module.IsRoot() {
		return nil
	}
	return s.RootOutputValues[addr.OutputValue.Name]
}

// SetOutputValue updates the value stored for the given output value if and
// only if it's a root module output value.
//
// All child module output values will just be silently ignored, because we
// don't store those here any more. (They live in a namedvals.State object
// hidden in the internals of Terraform Core.)
func (s *State) SetOutputValue(addr addrs.AbsOutputValue, value cty.Value, sensitive bool) {
	if !addr.Module.IsRoot() {
		return
	}
	s.RootOutputValues[addr.OutputValue.Name] = &OutputValue{
		Addr:      addr,
		Value:     value,
		Sensitive: sensitive,
	}
}

// RemoveOutputValue removes the record of a previously-stored output value.
func (s *State) RemoveOutputValue(addr addrs.AbsOutputValue) {
	if !addr.Module.IsRoot() {
		return
	}
	delete(s.RootOutputValues, addr.OutputValue.Name)
}

// EphemeralOutputValue returns the state for the output value with the given
// address, or nil if no such ephemeral output value is tracked in the state.
//
// Only root module output values are tracked in the state, so this always
// returns nil for output values in any other module.
func (s *State) EphemeralOutputValue(addr addrs.AbsOutputValue) *OutputValue {
	if !addr.Module.IsRoot() {
		return nil
	}
	return s.EphemeralRootOutputValues[addr.OutputValue.Name]
}

func (s *State) CombinedOutputValues() map[string]*OutputValue {
	combined := maps.Clone(s.RootOutputValues)
	maps.Copy(combined, s.EphemeralRootOutputValues)
	return combined
}

// SetEphemeralOutputValue updates the value stored for the given ephemeral
// output value if and only if it's a root module output value.
func (s *State) SetEphemeralOutputValue(addr addrs.AbsOutputValue, value cty.Value, sensitive bool) {
	if !addr.Module.IsRoot() {
		return
	}
	s.EphemeralRootOutputValues[addr.OutputValue.Name] = &OutputValue{
		Addr:      addr,
		Value:     value,
		Sensitive: sensitive,
		Ephemeral: true,
	}
}

// RemoveOutputValue removes the record of a previously-stored ephemeral output
// value.
func (s *State) RemoveEphemeralOutputValue(addr addrs.AbsOutputValue) {
	if !addr.Module.IsRoot() {
		return
	}
	delete(s.EphemeralRootOutputValues, addr.OutputValue.Name)
}

// ProviderAddrs returns a list of all of the provider configuration addresses
// referenced throughout the receiving state.
//
// The result is de-duplicated so that each distinct address appears only once.
func (s *State) ProviderAddrs() []addrs.AbsProviderConfig {
	if s == nil {
		return nil
	}

	m := map[string]addrs.AbsProviderConfig{}
	for _, ms := range s.Modules {
		for _, rc := range ms.Resources {
			m[rc.ProviderConfig.String()] = rc.ProviderConfig
		}
	}
	if len(m) == 0 {
		return nil
	}

	// This is mainly just so we'll get stable results for testing purposes.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := make([]addrs.AbsProviderConfig, len(keys))
	for i, key := range keys {
		ret[i] = m[key]
	}

	return ret
}

// ProviderRequirements returns a description of all of the providers that
// are required to work with the receiving state.
//
// Because the state does not track specific version information for providers,
// the requirements returned by this method will always be unconstrained.
// The result should usually be merged with a Requirements derived from the
// current configuration in order to apply some constraints.
func (s *State) ProviderRequirements() providerreqs.Requirements {
	configAddrs := s.ProviderAddrs()
	ret := make(providerreqs.Requirements, len(configAddrs))
	for _, configAddr := range configAddrs {
		ret[configAddr.Provider] = nil // unconstrained dependency
	}
	return ret
}

// PruneResourceHusks is a specialized method that will remove any Resource
// objects that do not contain any instances, even if they have an EachMode.
//
// This should generally be used only after a "terraform destroy" operation,
// to finalize the cleanup of the state. It is not correct to use this after
// other operations because if a resource has "count = 0" or "for_each" over
// an empty collection then we want to retain it in the state so that references
// to it, particularly in "strange" contexts like "terraform console", can be
// properly resolved.
//
// This method MUST NOT be called concurrently with other readers and writers
// of the receiving state.
func (s *State) PruneResourceHusks() {
	for _, m := range s.Modules {
		m.PruneResourceHusks()
		if len(m.Resources) == 0 && !m.Addr.IsRoot() {
			s.RemoveModule(m.Addr)
		}
	}
}

// SyncWrapper returns a SyncState object wrapping the receiver.
func (s *State) SyncWrapper() *SyncState {
	return &SyncState{
		state:    s,
		writable: true, // initially writable, becoming read-only once closed
	}
}

// MoveAbsResource moves the given src AbsResource's current state to the new
// dst address. This will panic if the src AbsResource does not exist in state,
// or if there is already a resource at the dst address. It is the caller's
// responsibility to verify the validity of the move (for example, that the src
// and dst are compatible types).
func (s *State) MoveAbsResource(src, dst addrs.AbsResource) {
	// verify that the src address exists and the dst address does not
	rs := s.Resource(src)
	if rs == nil {
		panic(fmt.Sprintf("no state for src address %s", src.String()))
	}

	ds := s.Resource(dst)
	if ds != nil {
		panic(fmt.Sprintf("dst resource %s already exists", dst.String()))
	}

	ms := s.Module(src.Module)
	ms.RemoveResource(src.Resource)

	// Remove the module if it is empty (and not root) after removing the
	// resource.
	if !ms.Addr.IsRoot() && ms.empty() {
		s.RemoveModule(src.Module)
	}

	// Update the address before adding it to the state
	rs.Addr = dst
	s.EnsureModule(dst.Module).Resources[dst.Resource.String()] = rs
}

// MaybeMoveAbsResource moves the given src AbsResource's current state to the
// new dst address. This function will succeed if both the src address does not
// exist in state and the dst address does; the return value indicates whether
// or not the move occurred. This function will panic if either the src does not
// exist or the dst does exist (but not both).
func (s *State) MaybeMoveAbsResource(src, dst addrs.AbsResource) bool {
	// Get the source and destinatation addresses from state.
	rs := s.Resource(src)
	ds := s.Resource(dst)

	// Normal case: the src exists in state, dst does not
	if rs != nil && ds == nil {
		s.MoveAbsResource(src, dst)
		return true
	}

	if rs == nil && ds != nil {
		// The source is not in state, the destination is. This is not
		// guaranteed to be idempotent since we aren't tracking exact moves, but
		// it's useful information for the caller.
		return false
	} else {
		panic("invalid move")
	}
}

// MoveAbsResourceInstance moves the given src AbsResourceInstance's current state to
// the new dst address. This will panic if the src AbsResourceInstance does not
// exist in state, or if there is already a resource at the dst address. It is
// the caller's responsibility to verify the validity of the move (for example,
// that the src and dst are compatible types).
func (s *State) MoveAbsResourceInstance(src, dst addrs.AbsResourceInstance) {
	srcInstanceState := s.ResourceInstance(src)
	if srcInstanceState == nil {
		panic(fmt.Sprintf("no state for src address %s", src.String()))
	}

	dstInstanceState := s.ResourceInstance(dst)
	if dstInstanceState != nil {
		panic(fmt.Sprintf("dst resource %s already exists", dst.String()))
	}

	srcResourceState := s.Resource(src.ContainingResource())
	srcProviderAddr := srcResourceState.ProviderConfig
	dstResourceAddr := dst.ContainingResource()

	// Remove the source resource instance from the module's state, and then the
	// module if empty.
	ms := s.Module(src.Module)
	ms.ForgetResourceInstanceAll(src.Resource)
	if !ms.Addr.IsRoot() && ms.empty() {
		s.RemoveModule(src.Module)
	}

	dstModule := s.EnsureModule(dst.Module)

	// See if there is already a resource we can add this instance to.
	dstResourceState := s.Resource(dstResourceAddr)
	if dstResourceState == nil {
		// If we're moving to an address without an index then that
		// suggests the user's intent is to establish both the
		// resource and the instance at the same time (since the
		// address covers both). If there's an index in the
		// target then allow creating the new instance here.
		dstModule.SetResourceProvider(
			dstResourceAddr.Resource,
			srcProviderAddr, // in this case, we bring the provider along as if we were moving the whole resource
		)
		dstResourceState = dstModule.Resource(dstResourceAddr.Resource)
	}

	dstResourceState.Instances[dst.Resource.Key] = srcInstanceState
}

// MaybeMoveAbsResourceInstance moves the given src AbsResourceInstance's
// current state to the new dst address. This function will succeed if both the
// src address does not exist in state and the dst address does; the return
// value indicates whether or not the move occured. This function will panic if
// either the src does not exist or the dst does exist (but not both).
func (s *State) MaybeMoveAbsResourceInstance(src, dst addrs.AbsResourceInstance) bool {
	// get the src and dst resource instances from state
	rs := s.ResourceInstance(src)
	ds := s.ResourceInstance(dst)

	// Normal case: the src exists in state, dst does not
	if rs != nil && ds == nil {
		s.MoveAbsResourceInstance(src, dst)
		return true
	}

	if rs == nil && ds != nil {
		// The source is not in state, the destination is. This is not
		// guaranteed to be idempotent since we aren't tracking exact moves, but
		// it's useful information.
		return false
	} else {
		panic("invalid move")
	}
}

// MoveModuleInstance moves the given src ModuleInstance's current state to the
// new dst address. This will panic if the src ModuleInstance does not
// exist in state, or if there is already a resource at the dst address. It is
// the caller's responsibility to verify the validity of the move.
func (s *State) MoveModuleInstance(src, dst addrs.ModuleInstance) {
	if src.IsRoot() || dst.IsRoot() {
		panic("cannot move to or from root module")
	}

	srcMod := s.Module(src)
	if srcMod == nil {
		panic(fmt.Sprintf("no state for src module %s", src.String()))
	}

	dstMod := s.Module(dst)
	if dstMod != nil {
		panic(fmt.Sprintf("dst module %s already exists in state", dst.String()))
	}

	s.RemoveModule(src)

	srcMod.Addr = dst
	s.EnsureModule(dst)
	s.Modules[dst.String()] = srcMod

	// Update any Resource's addresses.
	if srcMod.Resources != nil {
		for _, r := range srcMod.Resources {
			r.Addr.Module = dst
		}
	}
}

// MaybeMoveModuleInstance moves the given src ModuleInstance's current state to
// the new dst address. This function will succeed if both the src address does
// not exist in state and the dst address does; the return value indicates
// whether or not the move occured. This function will panic if either the src
// does not exist or the dst does exist (but not both).
func (s *State) MaybeMoveModuleInstance(src, dst addrs.ModuleInstance) bool {
	if src.IsRoot() || dst.IsRoot() {
		panic("cannot move to or from root module")
	}

	srcMod := s.Module(src)
	dstMod := s.Module(dst)

	// Normal case: the src exists in state, dst does not
	if srcMod != nil && dstMod == nil {
		s.MoveModuleInstance(src, dst)
		return true
	}

	if srcMod == nil || src.IsRoot() && dstMod != nil {
		// The source is not in state, the destination is. This is not
		// guaranteed to be idempotent since we aren't tracking exact moves, but
		// it's useful information.
		return false
	} else {
		panic("invalid move")
	}
}

// MoveModule takes a source and destination addrs.Module address, and moves all
// state Modules which are contained by the src address to the new address.
func (s *State) MoveModule(src, dst addrs.AbsModuleCall) {
	if src.Module.IsRoot() || dst.Module.IsRoot() {
		panic("cannot move to or from root module")
	}

	// Modules only exist as ModuleInstances in state, so we need to check each
	// state Module and see if it is contained by the src address to get a full
	// list of modules to move.
	var srcMIs []*Module
	for _, module := range s.Modules {
		if !module.Addr.IsRoot() {
			if src.Module.TargetContains(module.Addr) {
				srcMIs = append(srcMIs, module)
			}
		}
	}

	if len(srcMIs) == 0 {
		panic(fmt.Sprintf("no matching module instances found for src module %s", src.String()))
	}

	for _, ms := range srcMIs {
		newInst := make(addrs.ModuleInstance, len(ms.Addr))
		copy(newInst, ms.Addr)
		if ms.Addr.IsDeclaredByCall(src) {
			// Easy case: we just need to update the last step with the new name
			newInst[len(newInst)-1].Name = dst.Call.Name
		} else {
			// Trickier: this Module is a submodule. we need to find and update
			// only that appropriate step
			for s := range newInst {
				if newInst[s].Name == src.Call.Name {
					newInst[s].Name = dst.Call.Name
				}
			}
		}
		s.MoveModuleInstance(ms.Addr, newInst)
	}
}
