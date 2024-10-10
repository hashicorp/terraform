// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/states"
)

// State represents a previous run's state snapshot.
//
// Unlike [states.State] and its associates, State is an immutable data
// structure constructed to represent only the previous run state. It should
// not be modified after it's been constructed; results of planning or applying
// changes are represented in other ways inside the stacks language runtime.
type State struct {
	componentInstances collections.Map[stackaddrs.AbsComponentInstance, *componentInstanceState]
	outputs            map[stackaddrs.OutputValue]cty.Value
	inputs             map[stackaddrs.InputVariable]cty.Value

	// discardUnsupportedKeys is the set of state keys that we encountered
	// during decoding which are of types that are not supported by this
	// version of Terraform, if and only if they are of a type which is
	// specified as being discarded when unrecognized. We should emit
	// events during the apply phase to delete the objects associated with
	// these keys.
	discardUnsupportedKeys collections.Set[statekeys.Key]

	inputRaw map[string]*anypb.Any
}

// NewState constructs a new, empty state.
func NewState() *State {
	return &State{
		componentInstances:     collections.NewMap[stackaddrs.AbsComponentInstance, *componentInstanceState](),
		outputs:                make(map[stackaddrs.OutputValue]cty.Value),
		inputs:                 make(map[stackaddrs.InputVariable]cty.Value),
		discardUnsupportedKeys: statekeys.NewKeySet(),
		inputRaw:               nil,
	}
}

// RootInputVariables returns the values for the input variables currently in
// the state. An address that is in the map and maps to cty.NilVal is an
// ephemeral input, so it was present during the last operation but the value
// in unknown. Compared to an input variable not in the map at all, which
// indicates a new input variable that wasn't in the configuration during the
// last operation.
func (s *State) RootInputVariables() map[stackaddrs.InputVariable]cty.Value {
	return s.inputs
}

// RootInputVariable returns the input variable defined at the given address.
// If the second return value is true, then the value is present but is
// ephemeral and not known. If the first returned value is cty.NilVal and the
// second is false then the value isn't present in the state.
func (s *State) RootInputVariable(addr stackaddrs.InputVariable) cty.Value {
	return s.inputs[addr]
}

func (s *State) RootOutputValues() map[stackaddrs.OutputValue]cty.Value {
	return s.outputs
}

func (s *State) RootOutputValue(addr stackaddrs.OutputValue) cty.Value {
	return s.outputs[addr]
}

func (s *State) HasComponentInstance(addr stackaddrs.AbsComponentInstance) bool {
	return s.componentInstances.HasKey(addr)
}

// AllComponentInstances returns a set of addresses for all of the component
// instances that are tracked in the state.
//
// This includes both instances that were explicitly represented in the source
// raw state _and_ any that were missing but implied by a resource instance
// existing inside them. There should typically be an explicit component
// instance record tracked in raw state, but it can potentially be absent in
// exceptional cases such as if Terraform Core crashed partway through the
// previous run.
func (s *State) AllComponentInstances() collections.Set[stackaddrs.AbsComponentInstance] {
	var ret collections.Set[stackaddrs.AbsComponentInstance]
	if s.componentInstances.Len() == 0 {
		return ret
	}
	ret = collections.NewSet[stackaddrs.AbsComponentInstance]()
	for key := range s.componentInstances.All() {
		ret.Add(key)
	}
	return ret
}

// ComponentInstances returns the set of component instances that belong to the
// given component, or an empty set if no such component is tracked in the
// state.
//
// This will always be a subset of AllComponentInstances.
func (s *State) ComponentInstances(addr stackaddrs.AbsComponent) collections.Set[stackaddrs.ComponentInstance] {
	ret := collections.NewSet[stackaddrs.ComponentInstance]()
	for key := range s.componentInstances.All() {
		if key.Stack.String() != addr.Stack.String() {
			// Then
			continue
		}
		if key.Item.Component.Name != addr.Item.Name {
			continue
		}
		ret.Add(key.Item)
	}
	return ret
}

func (s *State) componentInstanceState(addr stackaddrs.AbsComponentInstance) *componentInstanceState {
	return s.componentInstances.Get(addr)
}

// DependenciesForComponent returns the list of components that are required by
// the given component instance, or an empty set if no such component instance
// is tracked in the state.
func (s *State) DependenciesForComponent(addr stackaddrs.AbsComponentInstance) collections.Set[stackaddrs.AbsComponent] {
	cs := s.componentInstanceState(addr)
	if cs == nil {
		return collections.NewSet[stackaddrs.AbsComponent]()
	}
	return cs.dependencies
}

// DependentsForComponent returns the list of components that are require the
// given component instance, or an empty set if no such component instance is
// tracked in the state.
func (s *State) DependentsForComponent(addr stackaddrs.AbsComponentInstance) collections.Set[stackaddrs.AbsComponent] {
	cs := s.componentInstanceState(addr)
	if cs == nil {
		return collections.NewSet[stackaddrs.AbsComponent]()
	}
	return cs.dependents
}

// ResultsForComponent returns the output values for the given component
// instance, or nil if no such component instance is tracked in the state.
func (s *State) ResultsForComponent(addr stackaddrs.AbsComponentInstance) map[addrs.OutputValue]cty.Value {
	cs := s.componentInstanceState(addr)
	if cs == nil {
		return nil
	}
	return cs.outputValues
}

// InputsForComponent returns the input values for the given component
// instance, or nil if no such component instance is tracked in the state.
func (s *State) InputsForComponent(addr stackaddrs.AbsComponentInstance) map[addrs.InputVariable]cty.Value {
	cs := s.componentInstanceState(addr)
	if cs == nil {
		return nil
	}
	return cs.inputVariables
}

// ComponentInstanceResourceInstanceObjects returns a set of addresses for
// all of the resource instance objects belonging to the component instance
// with the given address.
func (s *State) ComponentInstanceResourceInstanceObjects(addr stackaddrs.AbsComponentInstance) collections.Set[stackaddrs.AbsResourceInstanceObject] {
	var ret collections.Set[stackaddrs.AbsResourceInstanceObject]
	cs, ok := s.componentInstances.GetOk(addr)
	if !ok {
		return ret
	}
	ret = collections.NewSet[stackaddrs.AbsResourceInstanceObject]()
	for _, elem := range cs.resourceInstanceObjects.Elems {
		objKey := stackaddrs.AbsResourceInstanceObject{
			Component: addr,
			Item:      elem.Key,
		}
		ret.Add(objKey)
	}
	return ret
}

// AllResourceInstanceObjects returns a set of addresses for all of the resource
// instance objects that are tracked in the state, across all components.
func (s *State) AllResourceInstanceObjects() collections.Set[stackaddrs.AbsResourceInstanceObject] {
	ret := collections.NewSet[stackaddrs.AbsResourceInstanceObject]()
	for key, elem := range s.componentInstances.All() {
		componentAddr := key
		for _, elem := range elem.resourceInstanceObjects.Elems {
			objKey := stackaddrs.AbsResourceInstanceObject{
				Component: componentAddr,
				Item:      elem.Key,
			}
			ret.Add(objKey)
		}
	}
	return ret
}

// ResourceInstanceObjectSrc returns the source (i.e. still encoded) version of
// the resource instance object for the given address, or nil if no such
// object is tracked in the state.
func (s *State) ResourceInstanceObjectSrc(addr stackaddrs.AbsResourceInstanceObject) *states.ResourceInstanceObjectSrc {
	rios := s.resourceInstanceObjectState(addr)
	if rios == nil {
		return nil
	}
	return rios.src
}

// RequiredProviderInstances returns a description of all of the provider
// instance slots that are required to satisfy the resource instances
// belonging to the given component instance.
//
// See also stackeval.ComponentConfig.RequiredProviderInstances for a similar
// function that operates on the configuration of a component instance rather
// than the state of one.
func (s *State) RequiredProviderInstances(component stackaddrs.AbsComponentInstance) addrs.Set[addrs.RootProviderConfig] {
	state, ok := s.componentInstances.GetOk(component)
	if !ok {
		// Then we have no state for this component, which is fine.
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}

	providerInstances := addrs.MakeSet[addrs.RootProviderConfig]()
	for _, elem := range state.resourceInstanceObjects.Elems {
		providerInstances.Add(addrs.RootProviderConfig{
			Provider: elem.Value.providerConfigAddr.Provider,
			Alias:    elem.Value.providerConfigAddr.Alias,
		})
	}
	return providerInstances
}

func (s *State) resourceInstanceObjectState(addr stackaddrs.AbsResourceInstanceObject) *resourceInstanceObjectState {
	cs, ok := s.componentInstances.GetOk(addr.Component)
	if !ok {
		return nil
	}
	return cs.resourceInstanceObjects.Get(addr.Item)
}

// ComponentInstanceStateForModulesRuntime returns a [states.State]
// representation of the objects tracked for the given component instance.
//
// This produces only a very bare-bones [states.State] that should be
// sufficient for use as a prior state for the modules runtime's plan function
// to consider, but likely won't be of much other use.
func (s *State) ComponentInstanceStateForModulesRuntime(addr stackaddrs.AbsComponentInstance) *states.State {
	return states.BuildState(func(ss *states.SyncState) {
		objAddrs := s.ComponentInstanceResourceInstanceObjects(addr)
		for objAddr := range objAddrs.All() {
			rios := s.resourceInstanceObjectState(objAddr)

			if objAddr.Item.IsCurrent() {
				ss.SetResourceInstanceCurrent(
					objAddr.Item.ResourceInstance,
					rios.src, rios.providerConfigAddr,
				)
			} else {
				ss.SetResourceInstanceDeposed(
					objAddr.Item.ResourceInstance, objAddr.Item.DeposedKey,
					rios.src, rios.providerConfigAddr,
				)
			}
		}
	})
}

// RawKeysToDiscard returns a set of raw state keys that the apply phase should
// emit "delete" events for to remove objects from the raw state map that
// will no longer be relevant or meaningful after this plan is applied.
//
// Do not modify the returned set.
func (s *State) RawKeysToDiscard() collections.Set[statekeys.Key] {
	return s.discardUnsupportedKeys
}

// InputRaw returns the raw representation of state that this object was built
// from, or nil if this object wasn't constructed by decoding a protocol buffers
// representation.
//
// All callers of this method get the same map, so callers must not modify
// the map or anything reachable through it.
func (s *State) InputRaw() map[string]*anypb.Any {
	return s.inputRaw
}

func (s *State) addOutputValue(addr stackaddrs.OutputValue, value cty.Value) {
	s.outputs[addr] = value
}

func (s *State) addInputVariable(addr stackaddrs.InputVariable, value cty.Value) {
	s.inputs[addr] = value
}

func (s *State) ensureComponentInstanceState(addr stackaddrs.AbsComponentInstance) *componentInstanceState {
	if existing, ok := s.componentInstances.GetOk(addr); ok {
		return existing
	}
	s.componentInstances.Put(addr, &componentInstanceState{
		dependencies:            collections.NewSet[stackaddrs.AbsComponent](),
		dependents:              collections.NewSet[stackaddrs.AbsComponent](),
		outputValues:            make(map[addrs.OutputValue]cty.Value),
		inputVariables:          make(map[addrs.InputVariable]cty.Value),
		resourceInstanceObjects: addrs.MakeMap[addrs.AbsResourceInstanceObject, *resourceInstanceObjectState](),
	})
	return s.componentInstances.Get(addr)
}

func (s *State) addResourceInstanceObject(addr stackaddrs.AbsResourceInstanceObject, src *states.ResourceInstanceObjectSrc, providerConfigAddr addrs.AbsProviderConfig) {
	cs := s.ensureComponentInstanceState(addr.Component)

	cs.resourceInstanceObjects.Put(addr.Item, &resourceInstanceObjectState{
		src:                src,
		providerConfigAddr: providerConfigAddr,
	})
}

type componentInstanceState struct {
	// dependencies is the set of component instances that this component
	// depended on the last time it was updated.
	dependencies collections.Set[stackaddrs.AbsComponent]

	// dependents is a set of component instances that depended on this
	// component the last time it was updated.
	dependents collections.Set[stackaddrs.AbsComponent]

	// outputValues is a map from output value addresses to their values at
	// completion of the last apply operation.
	outputValues map[addrs.OutputValue]cty.Value

	// inputVariables is a map from input variable addresses to their values at
	// completion of the last apply operation.
	inputVariables map[addrs.InputVariable]cty.Value

	// resourceInstanceObjects is a map from resource instance object addresses
	// to their state.
	resourceInstanceObjects addrs.Map[addrs.AbsResourceInstanceObject, *resourceInstanceObjectState]
}

type resourceInstanceObjectState struct {
	src                *states.ResourceInstanceObjectSrc
	providerConfigAddr addrs.AbsProviderConfig
}
