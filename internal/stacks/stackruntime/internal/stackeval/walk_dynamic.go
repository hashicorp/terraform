// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
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
	walkDynamicObjectsInStack(ctx, walk, main.MainStack(ctx), phase, visit)
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
	for _, call := range stack.EmbeddedStackCalls(ctx) {
		call := call // separate symbol per loop iteration

		visit(ctx, walk, call)

		// We need to perform the whole expansion in an overall async task
		// because it involves evaluating for_each expressions, and one
		// stack call's for_each might depend on the results of another.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts, unknown := call.Instances(ctx, phase)

			// If unknown, then process the unknown instance and skip the rest.
			if unknown {
				inst := call.UnknownInstance(ctx, phase)
				visit(ctx, walk, inst)

				childStack := inst.CalledStack(ctx)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)
				return
			}

			// Otherwise, process the instances and their child stacks.
			for _, inst := range insts {
				visit(ctx, walk, inst)

				childStack := inst.CalledStack(ctx)
				walkDynamicObjectsInStack(ctx, walk, childStack, phase, visit)
			}
		})
	}

	// Next we're going to visit all the components and removed blocks and
	// execute each of the instances they represent. We might have component
	// and removed blocks that are target the same component address. This is
	// expected, but they shouldn't target the same instances.
	//
	// TODO: Check for this and don't evaluate problematic instances.
	//
	// In addition, we might have component and removed blocks that evaluate
	// to unknown instances. If this happens, we may have "unclaimed" instances.
	// This is normally an error, but if we an unknown values then they could
	// potentially be claimed once the unknown values are known.
	//
	// We'll allow both known removed and known component blocks to claim
	// anything they want. We'll then check for unclaimed instances and assign
	// them as being deferred but act as if they are part of whichever block
	// is unknown. If both are unknown, then all unclaimed instances will be
	// assigned to the component block and the removed block will do nothing.

	// We also need to visit and check all of the other declarations in
	// the current stack.
	for _, component := range stack.Components(ctx) {
		component := component // separate symbol per loop iteration
		visit(ctx, walk, component)

		// We need to perform the instance expansion in an overall async task
		// because it involves potentially evaluating a for_each expression.
		// and that might depend on data from elsewhere in the same stack.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts, unknown := component.Instances(ctx, phase)

			if unknown {
				// If the instances claimed by this component block are unknown,
				// then we'll check all the known instances and mark any that
				// are not claimed by an equivalent removed block as being
				// deferred until the foreach is known.

				// knownInstances are the instances that already exist either
				// because they are in the state or the plan.
				knownInstances := stack.KnownComponentInstances(component.Addr().Item, phase)
				if knownInstances.Len() == 0 {
					// If we have no known instances, then we'll make up an
					// unknown instance that will act as if it is being created
					// by the component block. This ensures the users still see
					// some feedback about this component even if we're not
					// doing anything with it this run block.
					//
					// If we have instances in state, the users will see
					// feedback about those instances so we don't need to do
					// this.
					unknownInstance := component.UnknownInstance(ctx, phase)
					visit(ctx, walk, unknownInstance)
					return
				}

				claimedInstances := collections.NewSet[stackaddrs.ComponentInstance]()
				removed := stack.Removed(ctx, component.Addr().Item)
				if removed != nil {
					// In this case we don't care about the unknown. If the
					// removed instances are unknown, then we'll mark everything
					// as being part of the component block. So, even if insts
					// comes back as unknown and hence empty, we still proceed
					// as normal.
					insts, _, _ := removed.Instances(ctx, phase)
					for key := range insts {
						claimedInstances.Add(stackaddrs.ComponentInstance{
							Component: component.Addr().Item,
							Key:       key,
						})
					}
				}

				for inst := range knownInstances.All() {
					if claimedInstances.Has(inst) {
						// Then this instance is claimed by the removed block.
						continue
					}

					// This instance is not claimed by the removed block, so
					// we'll mark it as being deferred until the foreach is
					// known.

					if inst.Key == addrs.WildcardKey {
						// If the key we retrieved is a wildcard key, then we'll
						// recreate a "proper" unknown instance as we'll
						// recompute a properly typed each value with this
						// function.
						inst := component.UnknownInstance(ctx, phase)
						visit(ctx, walk, inst)
					} else {
						// Otherwise, the key is a known key and the instance
						// actually does exist.
						inst := newComponentInstance(component, inst.Key, instances.RepetitionData{
							EachKey:   inst.Key.Value(),
							EachValue: cty.UnknownVal(cty.DynamicPseudoType),
						}, true)
						visit(ctx, walk, inst)
					}
				}

				return
			}

			for _, inst := range insts {
				visit(ctx, walk, inst)
			}
		})
	}
	for _, removed := range stack.Removeds(ctx) {
		removed := removed
		visit(ctx, walk, removed)

		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts, unknown, _ := removed.Instances(ctx, phase)
			if unknown {
				// If the instances claimed by this removed block are unknown,
				// then we'll check all the known instances and mark any that
				// are not claimed by an equivalent component block as being
				// removed by this block but as deferred until the foreach is
				// known.

				// knownInstances are the instances that already exist either
				// because they are in the state or the plan.
				knownInstances := stack.KnownComponentInstances(removed.Addr().Item, phase)

				claimedInstances := collections.NewSet[stackaddrs.ComponentInstance]()
				component := stack.Component(ctx, removed.Addr().Item)
				if component != nil {
					insts, unknown := component.Instances(ctx, phase)
					if unknown {
						// So both the for_each for the removed block and the
						// component block is unknown. In this case, we should
						// have gathered everything as being "updated" from the
						// component and we'll mark nothing as being removed.
						return
					}

					for key := range insts {
						claimedInstances.Add(stackaddrs.ComponentInstance{
							Component: removed.Addr().Item,
							Key:       key,
						})
					}
				}

				for inst := range knownInstances.All() {
					if claimedInstances.Has(inst) {
						// Then this instance is claimed by the component block.
						continue
					}

					// This instance is not claimed by the component block, so
					// we'll mark it as being removed by the removed block.
					inst := newRemovedInstance(removed, inst.Key, instances.RepetitionData{
						EachKey:   inst.Key.Value(),
						EachValue: cty.UnknownVal(cty.DynamicPseudoType),
					}, true)
					visit(ctx, walk, inst)
				}

				return
			}

			for _, inst := range insts {
				visit(ctx, walk, inst)
			}
		})
	}

	// Now, we'll do the rest of the declarations in the stack. These are
	// straightforward since we don't have to reconcile blocks that overlap.

	for _, provider := range stack.Providers(ctx) {
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
	for _, variable := range stack.InputVariables(ctx) {
		visit(ctx, walk, variable)
	}
	// TODO: Local values
	for _, output := range stack.OutputValues(ctx) {
		visit(ctx, walk, output)
	}

	// Finally we'll also check the stack itself, to deal with any problems
	// with the stack as a whole rather than individual declarations inside.
	visit(ctx, walk, stack)
}
