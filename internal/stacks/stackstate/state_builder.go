// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
)

// StateBuilder wraps State, and provides some write-only methods to update the
// state.
//
// This is generally used to build up a new state from scratch during tests.
type StateBuilder struct {
	state *State
}

func NewStateBuilder() *StateBuilder {
	return &StateBuilder{
		state: NewState(),
	}
}

// Build returns the state and invalidates the StateBuilder.
//
// You will get nil pointer exceptions if you attempt to use the builder after
// calling Build.
func (s *StateBuilder) Build() *State {
	ret := s.state
	s.state = nil
	return ret
}

// AddResourceInstance adds a resource instance to the state.
func (s *StateBuilder) AddResourceInstance(builder *ResourceInstanceBuilder) *StateBuilder {
	if builder.addr == nil || builder.src == nil || builder.providerAddr == nil {
		panic("ResourceInstanceBuilder is missing required fields")
	}
	s.state.addResourceInstanceObject(*builder.addr, builder.src, *builder.providerAddr)
	return s
}

// AddComponentInstance adds a component instance to the state.
func (s *StateBuilder) AddComponentInstance(builder *ComponentInstanceBuilder) *StateBuilder {
	component := s.state.ensureComponentInstanceState(builder.addr)
	component.outputValues = builder.outputValues
	component.inputVariables = builder.inputVariables

	for dep := range builder.dependencies.All() {
		component.dependencies.Add(dep)
	}
	for dep := range builder.dependents.All() {
		component.dependents.Add(dep)
	}
	return s
}

// AddOutput adds an output to the state.
func (s *StateBuilder) AddOutput(name string, value cty.Value) *StateBuilder {
	s.state.outputs[stackaddrs.OutputValue{Name: name}] = value
	return s
}

// AddInput adds an input variable to the state.
func (s *StateBuilder) AddInput(name string, value cty.Value) *StateBuilder {
	s.state.inputs[stackaddrs.InputVariable{Name: name}] = value
	return s
}

type ResourceInstanceBuilder struct {
	addr         *stackaddrs.AbsResourceInstanceObject
	src          *states.ResourceInstanceObjectSrc
	providerAddr *addrs.AbsProviderConfig
}

func NewResourceInstanceBuilder() *ResourceInstanceBuilder {
	return &ResourceInstanceBuilder{}
}

func (b *ResourceInstanceBuilder) SetAddr(addr stackaddrs.AbsResourceInstanceObject) *ResourceInstanceBuilder {
	b.addr = &addr
	return b
}

func (b *ResourceInstanceBuilder) SetResourceInstanceObjectSrc(src states.ResourceInstanceObjectSrc) *ResourceInstanceBuilder {
	b.src = &src
	return b
}

func (b *ResourceInstanceBuilder) SetProviderAddr(addr addrs.AbsProviderConfig) *ResourceInstanceBuilder {
	b.providerAddr = &addr
	return b
}

type ComponentInstanceBuilder struct {
	addr           stackaddrs.AbsComponentInstance
	dependencies   collections.Set[stackaddrs.AbsComponent]
	dependents     collections.Set[stackaddrs.AbsComponent]
	outputValues   map[addrs.OutputValue]cty.Value
	inputVariables map[addrs.InputVariable]cty.Value
}

func NewComponentInstanceBuilder(instance stackaddrs.AbsComponentInstance) *ComponentInstanceBuilder {
	return &ComponentInstanceBuilder{
		addr:           instance,
		dependencies:   collections.NewSet[stackaddrs.AbsComponent](),
		dependents:     collections.NewSet[stackaddrs.AbsComponent](),
		outputValues:   make(map[addrs.OutputValue]cty.Value),
		inputVariables: make(map[addrs.InputVariable]cty.Value),
	}
}

func (b *ComponentInstanceBuilder) AddDependency(addr stackaddrs.AbsComponent) *ComponentInstanceBuilder {
	b.dependencies.Add(addr)
	return b
}

func (b *ComponentInstanceBuilder) AddDependent(addr stackaddrs.AbsComponent) *ComponentInstanceBuilder {
	b.dependents.Add(addr)
	return b
}

func (b *ComponentInstanceBuilder) AddOutputValue(name string, value cty.Value) *ComponentInstanceBuilder {
	b.outputValues[addrs.OutputValue{Name: name}] = value
	return b
}

func (b *ComponentInstanceBuilder) AddInputVariable(name string, value cty.Value) *ComponentInstanceBuilder {
	b.inputVariables[addrs.InputVariable{Name: name}] = value
	return b
}
