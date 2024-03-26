// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
)

// StaticEvaler is implemented by types that participate in static
// evaluation phases, which currently includes [ValidatePhase] and [PlanPhase].
type StaticEvaler interface {
	Validatable
	Plannable
}

// walkDynamicObjects is a generic helper for visiting all of the "static
// objects" in scope for a particular [Main] object. "Static objects"
// essentially means the objects that are involved in the validation
// operation, which typically includes objects representing static
// configuration elements that haven't yet been expanded into their
// dynamic counterparts.
//
// The walk value stays constant throughout the walk, being passed to
// all visited objects. Visits can happen concurrently, so any methods
// offered by Output must be concurrency-safe.
//
// The Object type parameter should either be Validatable or Plannable
// depending on which of the two relevant evaluation phases this function
// is supposed to be driving.
func walkStaticObjects[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	main *Main,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj StaticEvaler),
) {
	walkStaticObjectsInStackConfig(ctx, walk, main.MainStackConfig(ctx), visit)
}

func walkStaticObjectsInStackConfig[Output any](
	ctx context.Context,
	walk *walkWithOutput[Output],
	stackConfig *StackConfig,
	visit func(ctx context.Context, walk *walkWithOutput[Output], obj StaticEvaler),
) {
	for _, obj := range stackConfig.InputVariables(ctx) {
		visit(ctx, walk, obj)
	}

	for _, obj := range stackConfig.OutputValues(ctx) {
		visit(ctx, walk, obj)
	}

	// TODO: All of the other static object types

	for _, obj := range stackConfig.Providers(ctx) {
		visit(ctx, walk, obj)
	}

	for _, obj := range stackConfig.Components(ctx) {
		visit(ctx, walk, obj)
	}

	for _, obj := range stackConfig.StackCalls(ctx) {
		visit(ctx, walk, obj)
	}

	for _, childCfg := range stackConfig.ChildConfigs(ctx) {
		walkStaticObjectsInStackConfig(ctx, walk, childCfg, visit)
	}
}
