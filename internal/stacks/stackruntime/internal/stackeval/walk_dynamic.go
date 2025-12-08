// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// DynamicEvaler is implemented by types that participate in dynamic
// evaluation phases, which currently includes [PlanPhase] and [ApplyPhase].
type DynamicEvaler interface {
	Plannable
	Applyable
}

// walkDynamicObjects is a generic helper for visiting all of the "dynamic
// objects" in scope for a particular [Main] object. "Dynamic objects"
// essentially means the objects that are involved in the plan and apply
// operations, which includes instances of objects that can expand using
// "count" or "for_each" arguments.
//
// The walk value stays constant throughout the walk, being passed to
// all visited objects. Visits can happen concurrently, so any methods
// offered by Output must be concurrency-safe.
//
// The type parameter Object should be either [Plannable] or [ApplyChecker]
// depending on which walk this call is intending to drive. All dynamic
// objects must implement both of those interfaces, although for many
// object types the logic is equivalent across both.
func walkDynamicObjects[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	main *Main,
	phase EvalPhase,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj DynamicEvaler),
) {
	walkDynamicObjectsInStack(ctx, walk, main.MainStack(), phase, visit)
}

func walkDynamicObjectsInStack[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	stack *Stack,
	phase EvalPhase,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj DynamicEvaler),
) {
	// We'll get the expansion of any child stack calls going first, so that
	// we can explore downstream stacks concurrently with this one. Each
	// stack call can represent zero or more child stacks that we'll analyze
	// by recursive calls to this function.
	for call := range stack.EmbeddedStackCalls() {
		walkEmbeddedStack(ctx, walk, stack, call, phase, visit)
	}
	for call := range stack.Removed().stackCalls {
		if stack.EmbeddedStackCall(call) != nil {
			continue
		}
		walkEmbeddedStack(ctx, walk, stack, call, phase, visit)
	}

	for component := range stack.Components() {
		walkComponent(ctx, walk, stack, component, phase, visit)
	}
	for component := range stack.Removed().components {
		if stack.Component(component) != nil {
			continue // then we processed this as part of the component stage
		}
		walkComponent(ctx, walk, stack, component, phase, visit)
	}

	// Now, we'll do the rest of the declarations in the stack. These are
	// straightforward since we don't have to reconcile blocks that overlap.

	for _, provider := range stack.Providers() {
		provider := provider // separate symbol per loop iteration

		visit(ctx, walk, provider)

		// We need to perform the instance expansion in an overall async
		// task because it involves potentially evaluating a for_each expression,
		// and that might depend on data from elsewhere in the same stack.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts, unknown := provider.Instances(ctx, phase)
			if unknown {
				// We use the unconfigured client for unknown instances of a
				// provider so there is nothing for us to do here.
				return
			}

			for _, inst := range insts {
				visit(ctx, walk, inst)
			}
		})
	}
	for _, variable := range stack.InputVariables() {
		visit(ctx, walk, variable)
	}
	for _, localValue := range stack.LocalValues() {
		visit(ctx, walk, localValue)
	}
	for _, output := range stack.OutputValues() {
		visit(ctx, walk, output)
	}

	// Finally we'll also check the stack itself, to deal with any problems
	// with the stack as a whole rather than individual declarations inside.
	visit(ctx, walk, stack)
}

// walkComponent just encapsulates the behaviour for visiting all the
// components in a stack. Components are more complicated than the other
// parts of a stack as they can be claimed by both component and removed blocks
// and both of these can evaluate to unknown.
//
// What we do here is we go through all the blocks within the configuration
// and visit them and mark the known instances as "claimed". We also want to
// find any blocks that evaluate to unknown. This block is then used in the
// final step where we'll search through any instances within the state that
// haven't been claimed and assign them to an unknown block, if one was found.
func walkComponent[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	stack *Stack,
	addr stackaddrs.Component,
	phase EvalPhase,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj DynamicEvaler)) {

	var unknownComponentBlock *Component
	var unknownRemovedComponentBlock *RemovedComponent

	var wg sync.WaitGroup
	var mutex sync.Mutex

	claimedInstances := collections.NewSet[stackaddrs.ComponentInstance]()

	component := stack.Component(addr)
	if component != nil {
		visit(ctx, walk, component) // first, just visit the component directly

		// then visit the component instances. we must do this in an async task as
		// we evaulate the for_each valuate within the instances call.

		wg.Add(1)
		walk.AsyncTask(ctx, func(ctx context.Context) {
			defer wg.Done()

			insts, unknown := component.Instances(ctx, phase)
			if unknown {
				unknownComponentBlock = component
				return
			}

			for key, inst := range insts {
				instAddr := stackaddrs.ComponentInstance{
					Component: addr,
					Key:       key,
				}

				mutex.Lock()
				if claimedInstances.Has(instAddr) {
					// this will be picked up as an error elsewhere, but
					// two blocks have claimed this instance so we'll just
					// allow whichever got their first to claim it and we'll
					// just skip it here.
					mutex.Unlock()
					continue
				}
				claimedInstances.Add(instAddr)
				mutex.Unlock()

				visit(ctx, walk, inst)
			}
		})
	}

	for _, block := range stack.Removed().components[addr] {
		visit(ctx, walk, block) // first, just visit the removed block directly

		wg.Add(1)
		walk.AsyncTask(ctx, func(ctx context.Context) {
			defer wg.Done()

			insts, unknown := block.InstancesFor(ctx, stack.addr, phase)
			if unknown {
				mutex.Lock()
				// we might have multiple removed blocks that evaluate to
				// unknown. if so, we'll just pick a random one that actually
				// gets assigned to handle any unclaimed instances.
				unknownRemovedComponentBlock = block
				mutex.Unlock()

				return
			}

			for _, inst := range insts {
				mutex.Lock()
				if claimedInstances.Has(inst.from.Item) {
					// this will be picked up as an error elsewhere, but
					// two blocks have claimed this instance so we'll just
					// allow whichever got their first to claim it and we'll
					// just skip it here.
					mutex.Unlock()
					continue
				}
				claimedInstances.Add(inst.from.Item)
				mutex.Unlock()

				visit(ctx, walk, inst)
			}
		})
	}

	// finally, we're going to look at the instances that are in state and
	// hopefully assign any unclaimed ones to an unknown block.

	walk.AsyncTask(ctx, func(ctx context.Context) {
		wg.Wait() // wait for all the other tasks to finish

		// if we have an unknown component block we want to make sure the
		// output says something about it. This means if we have unclaimed
		// instances then the component block will claim those, but if no
		// unclaimed instances exist we'll create a partial unknown component
		// instance that means the component block will appear in the plan
		// somewhere. This starts as true if there is no unknown component block
		// to make us not do anything by default for this case.
		unknownComponentBlockClaimedSomething := unknownComponentBlock == nil

		knownInstances := stack.KnownComponentInstances(addr, phase)
		for inst := range knownInstances {
			if claimedInstances.Has(inst) {
				// don't need the mutex any more since this will be fully
				// initialised when all the wait groups are finished.
				continue
			}

			// this is unclaimed, so we'll see if

			if unknownComponentBlock != nil {
				// then we have a component block to claim it so we make an
				// instance dynamically and off we go

				unknownComponentBlockClaimedSomething = true
				inst := unknownComponentBlock.UnknownInstance(ctx, inst.Key, phase)
				visit(ctx, walk, inst)
				continue
			}

			if unknownRemovedComponentBlock != nil {
				// then we didn't have an unknown component block, but we do
				// have an unknown removed component block to claim it

				from := stackaddrs.AbsComponentInstance{
					Stack: stack.addr,
					Item:  inst,
				}
				inst := unknownRemovedComponentBlock.UnknownInstance(ctx, from, phase)
				visit(ctx, walk, inst)

				continue
			}

			// then nothing claimed it - this is an error. We don't actually
			// raise this as an error here though (it will be caught elsewhere).

		}

		if !unknownComponentBlockClaimedSomething {
			// then we want to include the partial unknown component instance
			inst := unknownComponentBlock.UnknownInstance(ctx, addrs.WildcardKey, phase)
			visit(ctx, walk, inst)
		}
	})
}

// walkEmbeddedStack follows the pattern of walkComponent but for embedded
// stack calls rather than components.
func walkEmbeddedStack[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	stack *Stack,
	addr stackaddrs.StackCall,
	phase EvalPhase,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj DynamicEvaler)) {

	var unknownStackCall *StackCall
	var unknownRemovedStackCall *RemovedStackCall

	var wg sync.WaitGroup
	var mutex sync.Mutex

	claimedInstances := collections.NewSet[stackaddrs.StackInstance]()

	embeddedStack := stack.EmbeddedStackCall(addr)
	if embeddedStack != nil {
		visit(ctx, walk, embeddedStack)

		wg.Add(1)
		walk.AsyncTask(ctx, func(ctx context.Context) {
			defer wg.Done()

			insts, unknown := embeddedStack.Instances(ctx, phase)
			if unknown {
				unknownStackCall = embeddedStack
				return
			}

			for _, inst := range insts {
				instAddr := inst.CalledStackAddr()

				mutex.Lock()
				if claimedInstances.Has(instAddr) {
					mutex.Unlock()
					continue
				}
				claimedInstances.Add(instAddr)
				mutex.Unlock()

				visit(ctx, walk, inst)
				childStack := inst.Stack(ctx, phase)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)
			}
		})
	}

	for _, block := range stack.Removed().stackCalls[addr] {
		visit(ctx, walk, block)

		wg.Add(1)
		walk.AsyncTask(ctx, func(ctx context.Context) {
			defer wg.Done()

			insts, unknown := block.InstancesFor(ctx, stack.addr, phase)
			if unknown {
				mutex.Lock()
				unknownRemovedStackCall = block
				mutex.Unlock()

				return
			}

			for _, inst := range insts {
				mutex.Lock()
				if claimedInstances.Has(inst.from) {
					mutex.Unlock()
					continue
				}
				claimedInstances.Add(inst.from)
				mutex.Unlock()

				visit(ctx, walk, inst)
				childStack := inst.Stack(ctx, phase)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)
			}
		})
	}

	walk.AsyncTask(ctx, func(ctx context.Context) {
		wg.Wait()

		unknownStackCallClaimedSomething := unknownStackCall == nil

		knownStacks := stack.KnownEmbeddedStacks(addr, phase)
		for inst := range knownStacks {
			if claimedInstances.Has(inst) {
				continue
			}

			if unknownStackCall != nil {
				unknownStackCallClaimedSomething = true
				inst := unknownStackCall.UnknownInstance(ctx, inst[len(inst)-1].Key, phase)
				visit(ctx, walk, inst)
				childStack := inst.Stack(ctx, phase)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)

				continue
			}

			if unknownRemovedStackCall != nil {
				inst := unknownRemovedStackCall.UnknownInstance(ctx, inst, phase)
				visit(ctx, walk, inst)
				childStack := inst.Stack(ctx, phase)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)

				continue
			}
		}

		if !unknownStackCallClaimedSomething {
			// then we want to include the partial unknown component instance
			inst := unknownStackCall.UnknownInstance(ctx, addrs.WildcardKey, phase)
			visit(ctx, walk, inst)
			childStack := inst.Stack(ctx, phase)
			walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)
		}

	})

}
