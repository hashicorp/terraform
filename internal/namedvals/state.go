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
}

func NewState() *State {
	return &State{
		variables: newValues[addrs.InputVariable, addrs.AbsInputVariableInstance](),
		locals:    newValues[addrs.LocalValue, addrs.AbsLocalValue](),
		outputs:   newValues[addrs.OutputValue, addrs.AbsOutputValue](),
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

func (s *State) GetOutputValuesForModuleCall(parentAddr addrs.ModuleInstance, callAddr addrs.ModuleCall) addrs.Map[addrs.AbsOutputValue, cty.Value] {
	s.mu.Lock()
	defer s.mu.Unlock()

	// HACK: The "values" data structure isn't really designed to support
	// this operation, since it tries to be general over all different named
	// value address types but that makes it unable to generically handle
	// the problem of finding the module instance for a particular absolute
	// address. We'd need a ModuleInstance equivalent of
	// addrs.InPartialExpandedModule to achieve that, but our "Abs" address
	// types are all hand-written and predate Go having support for generic
	// types.
	//
	// This operation is just a stop-gap until we make the evaluator work
	// in a different way to handle placeholder values, so we'll accept it
	// being clunky and slow just as a checkpoint to make everything still
	// work similarly to how it used to, and then delete this function again
	// later once we can implement what we need using just
	// [State.GetOutputValue] by having the caller determine which output
	// values it should be asking for using the configuration.

	ret := addrs.MakeMap[addrs.AbsOutputValue, cty.Value]()
	all := s.outputs.GetExactResults()

	for _, elem := range all.Elems {
		outputMod := elem.Key.Module
		if outputMod.IsRoot() {
			// We cannot enumerate the root module output values with this
			// function, because the root module has no "call".
			continue
		}
		callingMod, call := outputMod.Call()
		if call != callAddr {
			continue
		}
		if !callingMod.Equal(parentAddr) {
			continue
		}

		// If we get here then the output value we're holding belongs to
		// one of the instances of the call indicated in this function's
		// arguments.
		ret.PutElement(elem)
	}

	return ret
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
