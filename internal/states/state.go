package states

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
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
// way to acheive this is to wrap the State in a SyncState and use the
// higher-level atomic operations supported by that type.
type State struct {
	// Modules contains the state for each module. The keys in this map are
	// an implementation detail and must not be used by outside callers.
	Modules map[string]*Module
}

// NewState constructs a minimal empty state, containing an empty root module.
func NewState() *State {
	modules := map[string]*Module{}
	modules[addrs.RootModuleInstance.String()] = NewModule(addrs.RootModuleInstance)
	return &State{
		Modules: modules,
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
	for _, ms := range s.Modules {
		if len(ms.Resources) != 0 {
			return false
		}
		if len(ms.OutputValues) != 0 {
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

// ModuleOutputs returns all outputs for the given module call under the
// parentAddr instance.
func (s *State) ModuleOutputs(parentAddr addrs.ModuleInstance, module addrs.ModuleCall) []*OutputValue {
	var os []*OutputValue
	for _, m := range s.Modules {
		// can't get outputs from the root module
		if m.Addr.IsRoot() {
			continue
		}

		parent, call := m.Addr.Call()
		// make sure this is a descendent in the correct path
		if !parentAddr.Equal(parent) {
			continue
		}

		// and check if this is the correct child
		if call.Name != module.Name {
			continue
		}

		for _, o := range m.OutputValues {
			os = append(os, o)
		}
	}

	return os
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

// AllManagedResourceInstanceObjectAddrs returns a set of addresses for all of
// the leaf resource instance objects associated with managed resources that
// are tracked in this state.
//
// This result is the set of objects that would be effectively "forgotten"
// (like "terraform state rm") if this state were totally discarded, such as
// by deleting a workspace. This function is intended only for reporting
// context in error messages, such as when we reject deleting a "non-empty"
// workspace as detected by s.HasManagedResourceInstanceObjects.
//
// The ordering of the result is meaningless but consistent. DeposedKey will
// be NotDeposed (the zero value of DeposedKey) for any "current" objects.
// This method is guaranteed to return at least one item if
// s.HasManagedResourceInstanceObjects returns true for the same state, and
// to return a zero-length slice if it returns false.
func (s *State) AllResourceInstanceObjectAddrs() []struct {
	Instance   addrs.AbsResourceInstance
	DeposedKey DeposedKey
} {
	if s == nil {
		return nil
	}

	// We use an unnamed return type here just because we currently have no
	// general need to return pairs of instance address and deposed key aside
	// from this method, and this method itself is only of marginal value
	// when producing some error messages.
	//
	// If that need ends up arising more in future then it might make sense to
	// name this as addrs.AbsResourceInstanceObject, although that would require
	// moving DeposedKey into the addrs package too.
	type ResourceInstanceObject = struct {
		Instance   addrs.AbsResourceInstance
		DeposedKey DeposedKey
	}
	var ret []ResourceInstanceObject

	for _, ms := range s.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}

			for instKey, is := range rs.Instances {
				instAddr := rs.Addr.Instance(instKey)
				if is.Current != nil {
					ret = append(ret, ResourceInstanceObject{instAddr, NotDeposed})
				}
				for deposedKey := range is.Deposed {
					ret = append(ret, ResourceInstanceObject{instAddr, deposedKey})
				}
			}
		}
	}

	sort.SliceStable(ret, func(i, j int) bool {
		objI, objJ := ret[i], ret[j]
		switch {
		case !objI.Instance.Equal(objJ.Instance):
			return objI.Instance.Less(objJ.Instance)
		default:
			return objI.DeposedKey < objJ.DeposedKey
		}
	})

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

// OutputValue returns the state for the output value with the given address,
// or nil if no such output value is tracked in the state.
func (s *State) OutputValue(addr addrs.AbsOutputValue) *OutputValue {
	ms := s.Module(addr.Module)
	if ms == nil {
		return nil
	}
	return ms.OutputValues[addr.OutputValue.Name]
}

// LocalValue returns the value of the named local value with the given address,
// or cty.NilVal if no such value is tracked in the state.
func (s *State) LocalValue(addr addrs.AbsLocalValue) cty.Value {
	ms := s.Module(addr.Module)
	if ms == nil {
		return cty.NilVal
	}
	return ms.LocalValues[addr.LocalValue.Name]
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
func (s *State) ProviderRequirements() getproviders.Requirements {
	configAddrs := s.ProviderAddrs()
	ret := make(getproviders.Requirements, len(configAddrs))
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
		state: s,
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
// or not the move occured. This function will panic if either the src does not
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

	// Update any OutputValues's addresses.
	if srcMod.OutputValues != nil {
		for _, ov := range srcMod.OutputValues {
			ov.Addr.Module = dst
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
