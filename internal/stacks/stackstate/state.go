// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"iter"

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
	root    *stackInstanceState
	outputs map[stackaddrs.OutputValue]cty.Value
	inputs  map[stackaddrs.InputVariable]cty.Value

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
		root:                   newStackInstanceState(stackaddrs.RootStackInstance),
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
	stack := s.root.getDescendent(addr.Stack)
	if stack == nil {
		return false
	}
	return stack.getComponentInstance(addr.Item) != nil
}

func (s *State) HasStackInstance(addr stackaddrs.StackInstance) bool {
	stack := s.root.getDescendent(addr)
	return stack != nil
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
func (s *State) AllComponentInstances() iter.Seq[stackaddrs.AbsComponentInstance] {
	return func(yield func(stackaddrs.AbsComponentInstance) bool) {
		s.root.iterate(yield)
	}
}

// ComponentInstances returns the set of component instances that belong to the
// given component, or an empty set if no such component is tracked in the
// state.
//
// This will always be a subset of AllComponentInstances.
func (s *State) ComponentInstances(addr stackaddrs.AbsComponent) iter.Seq[stackaddrs.ComponentInstance] {
	return func(yield func(stackaddrs.ComponentInstance) bool) {
		target := s.root.getDescendent(addr.Stack)
		if target == nil {
			return
		}

		for key := range target.components[addr.Item] {
			yield(stackaddrs.ComponentInstance{
				Component: addr.Item,
				Key:       key,
			})
		}
	}
}

// StackInstances returns the set of known stack instances for the given stack
// call.
func (s *State) StackInstances(call stackaddrs.AbsStackCall) iter.Seq[stackaddrs.StackInstance] {
	return func(yield func(stackaddrs.StackInstance) bool) {
		target := s.root.getDescendent(call.Stack)
		if target == nil {
			return
		}

		for _, stack := range target.children[call.Item.Name] {
			yield(stack.address)
		}
	}
}

func (s *State) componentInstanceState(addr stackaddrs.AbsComponentInstance) *componentInstanceState {
	target := s.root.getDescendent(addr.Stack)
	if target == nil {
		return nil
	}
	return target.getComponentInstance(addr.Item)
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

type IdentitySrc struct {
	IdentitySchemaVersion uint64
	IdentityJSON          []byte
}

// IdentitiesForComponent returns the identity values for the given component
// instance, or nil if no such component instance is tracked in the state.
func (s *State) IdentitiesForComponent(addr stackaddrs.AbsComponentInstance) map[*addrs.AbsResourceInstanceObject]IdentitySrc {
	cs := s.componentInstanceState(addr)
	if cs == nil {
		return nil
	}

	res := make(map[*addrs.AbsResourceInstanceObject]IdentitySrc)
	for _, rio := range cs.resourceInstanceObjects.Elements() {
		res[&rio.Key] = IdentitySrc{
			IdentitySchemaVersion: rio.Value.src.IdentitySchemaVersion,
			IdentityJSON:          rio.Value.src.IdentityJSON,
		}
	}

	return res
}

// ComponentInstanceResourceInstanceObjects returns a set of addresses for
// all of the resource instance objects belonging to the component instance
// with the given address.
func (s *State) ComponentInstanceResourceInstanceObjects(addr stackaddrs.AbsComponentInstance) collections.Set[stackaddrs.AbsResourceInstanceObject] {
	var ret collections.Set[stackaddrs.AbsResourceInstanceObject]
	cs := s.componentInstanceState(addr)
	if cs == nil {
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
	state := s.componentInstanceState(component)
	if state == nil {
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
	cs := s.componentInstanceState(addr.Component)
	if cs == nil {
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
	current := s.root
	for _, step := range addr.Stack {
		next := current.getChild(step)
		if next == nil {
			next = newStackInstanceState(append(current.address, step))

			children, ok := current.children[step.Name]
			if !ok {
				children = make(map[addrs.InstanceKey]*stackInstanceState)
			}
			children[step.Key] = next
			current.children[step.Name] = children
		}
		current = next
	}

	component := current.getComponentInstance(addr.Item)
	if component == nil {
		component = &componentInstanceState{
			dependencies:            collections.NewSet[stackaddrs.AbsComponent](),
			dependents:              collections.NewSet[stackaddrs.AbsComponent](),
			outputValues:            make(map[addrs.OutputValue]cty.Value),
			inputVariables:          make(map[addrs.InputVariable]cty.Value),
			resourceInstanceObjects: addrs.MakeMap[addrs.AbsResourceInstanceObject, *resourceInstanceObjectState](),
		}

		components, ok := current.components[addr.Item.Component]
		if !ok {
			components = make(map[addrs.InstanceKey]*componentInstanceState)
		}
		components[addr.Item.Key] = component
		current.components[addr.Item.Component] = components
	}
	return component
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

type stackInstanceState struct {
	address    stackaddrs.StackInstance
	components map[stackaddrs.Component]map[addrs.InstanceKey]*componentInstanceState
	children   map[string]map[addrs.InstanceKey]*stackInstanceState
}

func newStackInstanceState(address stackaddrs.StackInstance) *stackInstanceState {
	return &stackInstanceState{
		address:    address,
		components: make(map[stackaddrs.Component]map[addrs.InstanceKey]*componentInstanceState),
		children:   make(map[string]map[addrs.InstanceKey]*stackInstanceState),
	}
}

func (s *stackInstanceState) getDescendent(stack stackaddrs.StackInstance) *stackInstanceState {
	if len(stack) == 0 {
		return s
	}

	next := s.getChild(stack[0])
	if next == nil {
		return nil
	}
	return next.getDescendent(stack[1:])
}

func (s *stackInstanceState) getChild(step stackaddrs.StackInstanceStep) *stackInstanceState {
	stacks, ok := s.children[step.Name]
	if !ok {
		return nil
	}
	return stacks[step.Key]
}

func (s *stackInstanceState) getComponentInstance(component stackaddrs.ComponentInstance) *componentInstanceState {
	components, ok := s.components[component.Component]
	if !ok {
		return nil
	}
	return components[component.Key]
}

func (s *stackInstanceState) iterate(yield func(stackaddrs.AbsComponentInstance) bool) bool {
	for component, components := range s.components {
		for key := range components {
			proceed := yield(stackaddrs.AbsComponentInstance{
				Stack: s.address,
				Item: stackaddrs.ComponentInstance{
					Component: component,
					Key:       key,
				},
			})
			if !proceed {
				return false
			}
		}
	}

	for _, children := range s.children {
		for _, child := range children {
			if !child.iterate(yield) {
				return false
			}
		}
	}

	return true
}
