// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
)

// Removed encapsulates the somewhat complicated logic for tracking and
// managing the removed block instances in a given stack.
//
// The Removed block does actually capture the entire tree of removed blocks
// in a single instance via the children field. Each Stack has a reference to
// its Removed instance, from which it can access all of its children.
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

// validateMissingInstanceAgainstRemovedBlocks matches the function of the same
// name defined on Stack.
//
// This function should only ever be called from inside that function and it
// performs the same purpose except it exclusively looks for orphaned blocks
// with the children.
//
// This function assumes all the checks made in the equivalent function in Stack
// have been completed, so again (!!!) it should only be called from within
// the other function.
func (removed *Removed) validateMissingInstanceAgainstRemovedBlocks(ctx context.Context, addr stackaddrs.StackInstance, target stackaddrs.AbsComponentInstance, phase EvalPhase) (*stackconfig.Removed, *stackconfig.Component) {

	// we're just jumping directly into checking the children, the removed
	// stack calls should have already been checked by the function on
	// Stack.

	if len(target.Stack) == 0 {

		if components, ok := removed.components[target.Item.Component]; ok {
			for _, component := range components {
				insts, _ := component.InstancesFor(ctx, addr, phase)
				for _, inst := range insts {
					if inst.from.Item.Key == target.Item.Key {
						// then we have actually found it! this is a removed
						// block that targets the target address, but isn't
						// in any stacks.
						return inst.call.config.config, nil
					}
				}
			}
		}

		return nil, nil // we found no potential blocks
	}

	// otherwise, we'll keep looking!

	// first, we'll check to see if we have a removed block targeting
	// the entire stack.

	next := target.Stack[0]
	rest := stackaddrs.AbsComponentInstance{
		Stack: target.Stack[1:],
		Item:  target.Item,
	}

	if calls, ok := removed.stackCalls[stackaddrs.StackCall{Name: next.Name}]; ok {
		for _, call := range calls {
			insts, _ := call.InstancesFor(ctx, append(addr, next), phase)
			for _, inst := range insts {
				stack := inst.Stack(ctx, phase)

				// now, hand the search back over to the stack to check if
				// the target instance is actually claimed by this removed
				// stack.
				removed, component := stack.validateMissingInstanceAgainstRemovedBlocks(ctx, rest, phase)
				if removed != nil || component != nil {
					// if we found any match, then return this removed block
					// as the original source
					return call.config.config, nil
				}
			}
		}

	}

	// finally, we'll keep going through the children of the next one.

	if child, ok := removed.children[next.Name]; ok {
		return child.validateMissingInstanceAgainstRemovedBlocks(ctx, append(addr, next), rest, phase)
	}

	return nil, nil
}
