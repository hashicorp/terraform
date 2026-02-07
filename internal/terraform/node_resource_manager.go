// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/tfdiags"

// ResourceState is an interface that defines the contract for executing
// a resource state. It takes a context, a node, and resource data as input
// and returns a new resource state and any diagnostics that occurred during
// the execution.
type ResourceState[T any] interface {
	Execute(ctx EvalContext, node T, data *ResourceData) (ResourceState[T], tfdiags.Diagnostics)
}

// ResourceStateManager is a generic state manager for resource instances
// It manages the state of a resource instance and its transitions
// between different states.
type ResourceStateManager[T any] struct {
	node  T
	data  *ResourceData
	hooks []func(ResourceState[T], *ResourceStateManager[T])
}

func NewResourceStateManager[T any](node T) *ResourceStateManager[T] {
	return &ResourceStateManager[T]{
		node:  node,
		data:  &ResourceData{},
		hooks: []func(ResourceState[T], *ResourceStateManager[T]){},
	}
}

func (m *ResourceStateManager[T]) AddHook(hook func(ResourceState[T], *ResourceStateManager[T])) {
	m.hooks = append(m.hooks, hook)
}

func (m *ResourceStateManager[T]) Execute(start ResourceState[T], ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Start with initial state
	currentState := start

	// Execute state transitions until completion or error
	for currentState != nil && !diags.HasErrors() {
		for _, hook := range m.hooks {
			hook(currentState, m)
		}
		var stateDiags tfdiags.Diagnostics
		currentState, stateDiags = currentState.Execute(ctx, m.node, m.data)
		diags = diags.Append(stateDiags)
	}

	return diags
}
