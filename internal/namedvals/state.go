// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package namedvals

import (
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// State is the main type in this package, representing the current state of
// evaluation that can be mutated as Terraform Core visits different graph
// nodes and then queried to find values that were already resolved earlier
// in the graph walk.
//
// Instances of this type are concurrency-safe so callers do not need to
// implement their own locking when reading and writing from named value
// state.
type State struct {
	mu sync.Mutex

	variables inputVariableValues
	locals    localValues
	outputs   outputValues
	lists     addrs.Map[addrs.Resource, addrs.Map[addrs.AbsResourceInstance, cty.Value]]
}

func NewState() *State {
	return &State{
		variables: newValues[addrs.InputVariable, addrs.AbsInputVariableInstance](),
		locals:    newValues[addrs.LocalValue, addrs.AbsLocalValue](),
		outputs:   newValues[addrs.OutputValue, addrs.AbsOutputValue](),
		lists:     addrs.MakeMap[addrs.Resource, addrs.Map[addrs.AbsResourceInstance, cty.Value]](),
	}
}

func (s *State) SetInputVariableValue(addr addrs.AbsInputVariableInstance, val cty.Value) {
	s.mu.Lock()
	s.variables.SetExactResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetInputVariableValue(addr addrs.AbsInputVariableInstance) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.variables.GetExactResult(addr)
}

func (s *State) HasInputVariableValue(addr addrs.AbsInputVariableInstance) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.variables.HasExactResult(addr)
}

func (s *State) SetInputVariablePlaceholder(addr addrs.InPartialExpandedModule[addrs.InputVariable], val cty.Value) {
	s.mu.Lock()
	s.variables.SetPlaceholderResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetInputVariablePlaceholder(addr addrs.InPartialExpandedModule[addrs.InputVariable]) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.variables.GetPlaceholderResult(addr)
}

func (s *State) SetLocalValue(addr addrs.AbsLocalValue, val cty.Value) {
	s.mu.Lock()
	s.locals.SetExactResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetLocalValue(addr addrs.AbsLocalValue) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locals.GetExactResult(addr)
}

func (s *State) HasLocalValue(addr addrs.AbsLocalValue) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locals.HasExactResult(addr)
}

func (s *State) SetLocalValuePlaceholder(addr addrs.InPartialExpandedModule[addrs.LocalValue], val cty.Value) {
	s.mu.Lock()
	s.locals.SetPlaceholderResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetLocalValuePlaceholder(addr addrs.InPartialExpandedModule[addrs.LocalValue]) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locals.GetPlaceholderResult(addr)
}

func (s *State) SetOutputValue(addr addrs.AbsOutputValue, val cty.Value) {
	s.mu.Lock()
	s.outputs.SetExactResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetOutputValue(addr addrs.AbsOutputValue) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputs.GetExactResult(addr)
}

func (s *State) HasOutputValue(addr addrs.AbsOutputValue) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputs.HasExactResult(addr)
}

func (s *State) SetOutputValuePlaceholder(addr addrs.InPartialExpandedModule[addrs.OutputValue], val cty.Value) {
	s.mu.Lock()
	s.outputs.SetPlaceholderResult(addr, val)
	s.mu.Unlock()
}

func (s *State) GetOutputValuePlaceholder(addr addrs.InPartialExpandedModule[addrs.OutputValue]) cty.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputs.GetPlaceholderResult(addr)
}

func (s *State) SetResourceListInstance(addr addrs.AbsResource, key addrs.InstanceKey, val cty.Value) {
	s.mu.Lock()
	if !s.lists.Has(addr.Resource) {
		s.lists.Put(addr.Resource, addrs.MakeMap[addrs.AbsResourceInstance, cty.Value]())
	}
	s.lists.Get(addr.Resource).Put(addr.Instance(key), val)
	s.mu.Unlock()
}

func (s *State) GetResourceListInstances(addr addrs.AbsResource) addrs.Map[addrs.AbsResourceInstance, cty.Value] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lists.Has(addr.Resource) {
		return addrs.MakeMap[addrs.AbsResourceInstance, cty.Value]()
	}
	insts := s.lists.Get(addr.Resource)
	return insts
}

func (s *State) AllResourceListInstances() addrs.Map[addrs.Resource, addrs.Map[addrs.AbsResourceInstance, cty.Value]] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lists
}
