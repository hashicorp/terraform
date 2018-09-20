package states

import (
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
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
		panic("attempted to remote root module")
	}

	delete(s.Modules, addr.String())
}

// RootModule is a convenient alias for Module(addrs.RootModuleInstance).
func (s *State) RootModule() *Module {
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

// HasResources returns true if there is at least one resource (of any mode)
// present in the receiving state.
func (s *State) HasResources() bool {
	if s == nil {
		return false
	}
	for _, ms := range s.Modules {
		if len(ms.Resources) > 0 {
			return true
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

// SyncWrapper returns a SyncState object wrapping the receiver.
func (s *State) SyncWrapper() *SyncState {
	return &SyncState{
		state: s,
	}
}
