// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"github.com/hashicorp/terraform/internal/addrs"
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
