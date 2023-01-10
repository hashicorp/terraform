package collections

import "github.com/hashicorp/terraform/internal/plans"

type ProcessKey[Output any] func(key string) (Output, plans.Action)

func TransformMap[Input, Output any](before, after map[string]Input, process ProcessKey[Output]) (map[string]Output, plans.Action) {
	current := plans.NoOp
	if before != nil && after == nil {
		current = plans.Delete
	}
	if before == nil && after != nil {
		current = plans.Create
	}

	elements := make(map[string]Output)
	for key := range before {
		var action plans.Action
		elements[key], action = process(key)
		current = CompareActions(current, action)
	}

	for key := range after {
		if _, ok := elements[key]; ok {
			// Then we've already processed this key in the before.
			continue
		}
		var action plans.Action
		elements[key], action = process(key)
		current = CompareActions(current, action)
	}

	return elements, current
}
