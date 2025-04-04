// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Removed encapsulates the somewhat complicated logic for tracking and
// managing the removed block instances in a given stack.
//
// All addresses within Removed are relative to the current stack.
type Removed struct {
	stackCallComponents collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent]
	localComponents     map[stackaddrs.Component][]*RemovedComponent
	embeddedStackCalls  collections.Map[stackaddrs.Stack, []*RemovedStackCall]
	localStackCalls     map[stackaddrs.StackCall][]*RemovedStackCall
}

func newRemoved(localComponents map[stackaddrs.Component][]*RemovedComponent,
	stackCallComponents collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent],
	localStackCalls map[stackaddrs.StackCall][]*RemovedStackCall,
	embeddedStackCalls collections.Map[stackaddrs.Stack, []*RemovedStackCall]) *Removed {
	return &Removed{
		stackCallComponents: stackCallComponents,
		localComponents:     localComponents,
		localStackCalls:     localStackCalls,
		embeddedStackCalls:  embeddedStackCalls,
	}
}

// ForStackCall returns all removed component blocks that target the given
// stack call. The addresses are transformed to be relative to the stack
// created by the stack call.
func (r *Removed) ForStackCall(addr stackaddrs.StackCall) (collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent], collections.Map[stackaddrs.Stack, []*RemovedStackCall]) {
	components := collections.NewMap[stackaddrs.ConfigComponent, []*RemovedComponent]()
	for target, blocks := range r.stackCallComponents.All() {
		step := target.Stack[0]
		rest := target.Stack[1:]

		if step.Name != addr.Name {
			continue
		}

		components.Put(stackaddrs.ConfigComponent{
			Stack: rest,
			Item:  target.Item,
		}, blocks)
	}
	stackCalls := collections.NewMap[stackaddrs.Stack, []*RemovedStackCall]()
	for target, blocks := range r.embeddedStackCalls.All() {
		step := target[0]
		rest := target[1:]

		if step.Name != addr.Name {
			continue
		}

		stackCalls.Put(rest, blocks)
	}
	return components, stackCalls
}
