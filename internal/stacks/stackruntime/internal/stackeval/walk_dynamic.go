// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
)

// DynamicEvaler is implemented by types that participate in dynamic
// evaluation phases, which currently includes [PlanPhase] and [ApplyPhase].
type DynamicEvaler interface {
	Plannable
	ApplyChecker
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
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj DynamicEvaler),
) {
	walkDynamicObjectsInStack(ctx, walk, main.MainStack(ctx), visit)
}

func walkDynamicObjectsInStack[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	stack *Stack,
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
			insts := call.Instances(ctx, PlanPhase)
			for _, inst := range insts {
				visit(ctx, walk, inst)

				childStack := inst.CalledStack(ctx)
				walkDynamicObjectsInStack(ctx, walk, childStack, visit)
			}
		})
	}

	// We also need to visit and check all of the other declarations in
	// the current stack.

	for _, component := range stack.Components(ctx) {
		component := component // separate symbol per loop iteration

		visit(ctx, walk, component)

		// We need to perform the instance expansion in an overall async task
		// because it involves potentially evaluating a for_each expression.
		// and that might depend on data from elsewhere in the same stack.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts := component.Instances(ctx, PlanPhase)
			for _, inst := range insts {
				visit(ctx, walk, inst)
			}
		})
	}
	for _, provider := range stack.Providers(ctx) {
		provider := provider // separate symbol per loop iteration

		visit(ctx, walk, provider)

		// We need to perform the instance expansion in an overall async
		// task because it involves potentially evaluating a for_each expression,
		// and that might depend on data from elsewhere in the same stack.
		walk.AsyncTask(ctx, func(ctx context.Context) {
			insts := provider.Instances(ctx, PlanPhase)
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
