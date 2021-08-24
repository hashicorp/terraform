package instances

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// Set is a set of instances, intended mainly for the return value of
// Expander.AllInstances, where it therefore represents all of the module
// and resource instances known to the expander.
type Set struct {
	// Set currently really just wraps Expander with a reduced API that
	// only supports lookups, to make it clear that a holder of a Set should
	// not be modifying the expander any further.
	exp *Expander
}

// HasModuleInstance returns true if and only if the set contains the module
// instance with the given address.
func (s Set) HasModuleInstance(want addrs.ModuleInstance) bool {
	return s.exp.knowsModuleInstance(want)
}

// HasModuleCall returns true if and only if the set contains the module
// call with the given address, even if that module call has no instances.
func (s Set) HasModuleCall(want addrs.AbsModuleCall) bool {
	return s.exp.knowsModuleCall(want)
}

// HasResourceInstance returns true if and only if the set contains the resource
// instance with the given address.
// TODO:
func (s Set) HasResourceInstance(want addrs.AbsResourceInstance) bool {
	return s.exp.knowsResourceInstance(want)
}

// HasResource returns true if and only if the set contains the resource with
// the given address, even if that resource has no instances.
// TODO:
func (s Set) HasResource(want addrs.AbsResource) bool {
	return s.exp.knowsResource(want)
}

// InstancesForModule returns all of the module instances that correspond with
// the given static module path.
//
// If there are multiple module calls in the path that have repetition enabled
// then the result is the full expansion of all combinations of all of their
// declared instance keys.
func (s Set) InstancesForModule(modAddr addrs.Module) []addrs.ModuleInstance {
	return s.exp.ExpandModule(modAddr)
}
