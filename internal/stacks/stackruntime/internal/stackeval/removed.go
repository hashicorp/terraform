// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"sync"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Removed encapsulates the somewhat complicated logic for tracking and
// managing the removed block instances in a given stack.
type Removed struct {
	sync.Mutex

	components map[stackaddrs.Component][]*RemovedComponent
	stackCalls map[stackaddrs.StackCall][]*RemovedStackCall

	children map[string]*Removed
}

func newRemoved() *Removed {
	return &Removed{
		components: make(map[stackaddrs.Component][]*RemovedComponent),
		stackCalls: make(map[stackaddrs.StackCall][]*RemovedStackCall),
		children:   make(map[string]*Removed),
	}
}

func (removed *Removed) Get(addr stackaddrs.ConfigStackCall) *Removed {
	if len(addr.Stack) == 0 {
		return removed.Next(addr.Item.Name)
	}
	return removed.Next(addr.Stack[0].Name).Get(stackaddrs.ConfigStackCall{
		Stack: addr.Stack[1:],
		Item:  addr.Item,
	})
}

func (removed *Removed) Next(step string) *Removed {
	removed.Lock()
	defer removed.Unlock()

	next := removed.children[step]
	if next == nil {
		next = newRemoved()
		removed.children[step] = next
	}
	return next
}

func (removed *Removed) AddComponent(addr stackaddrs.ConfigComponent, components []*RemovedComponent) {
	if len(addr.Stack) == 0 {
		removed.components[addr.Item] = append(removed.components[addr.Item], components...)
		return
	}
	removed.Next(addr.Stack[0].Name).AddComponent(stackaddrs.ConfigComponent{
		Stack: addr.Stack[1:],
		Item:  addr.Item,
	}, components)
}

func (removed *Removed) AddStackCall(addr stackaddrs.ConfigStackCall, stackCalls []*RemovedStackCall) {
	if len(addr.Stack) == 0 {
		removed.stackCalls[addr.Item] = append(removed.stackCalls[addr.Item], stackCalls...)
		return
	}
	removed.Next(addr.Stack[0].Name).AddStackCall(stackaddrs.ConfigStackCall{
		Stack: addr.Stack[1:],
		Item:  addr.Item,
	}, stackCalls)
}
