package states

import (
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
// and that reads only occur when no write is in progress.
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

// Module returns the state for the module with the given address, or nil if
// the requested module is not tracked in the state.
func (s *State) Module(addr addrs.ModuleInstance) *Module {
	return s.Modules[addr.String()]
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
