package collections

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

type ProcessKey func(key string) computed.Diff

func TransformMap[Input any](before, after map[string]Input, process ProcessKey) (map[string]computed.Diff, plans.Action) {
	current := plans.NoOp
	if before != nil && after == nil {
		current = plans.Delete
	}
	if before == nil && after != nil {
		current = plans.Create
	}

	elements := make(map[string]computed.Diff)
	for key := range before {
		elements[key] = process(key)
		current = CompareActions(current, elements[key].Action)
	}

	for key := range after {
		if _, ok := elements[key]; ok {
			// Then we've already processed this key in the before.
			continue
		}
		elements[key] = process(key)
		current = CompareActions(current, elements[key].Action)
	}

	return elements, current
}
