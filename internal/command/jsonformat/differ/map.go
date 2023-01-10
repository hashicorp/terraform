package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"

	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) computeAttributeDiffAsMap(elementType cty.Type) computed.Diff {
	current := change.getDefaultActionForIteration()
	elements := make(map[string]computed.Diff)
	change.processMap(func(key string, value Change) {
		element := value.computeDiffForType(elementType)
		elements[key] = element
		current = compareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.Map(elements), current, change.replacePath())
}

func (change Change) computeAttributeDiffAsNestedMap(attributes map[string]*jsonprovider.Attribute) computed.Diff {
	current := change.getDefaultActionForIteration()
	elements := make(map[string]computed.Diff)
	change.processMap(func(key string, value Change) {
		element := value.computeDiffForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements[key] = element
		current = compareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.Map(elements), current, change.replacePath())
}

func (change Change) computeBlockDiffsAsMap(block *jsonprovider.Block) ([]computed.Diff, plans.Action) {
	current := change.getDefaultActionForIteration()
	var elements []computed.Diff
	change.processMap(func(key string, value Change) {
		element := value.ComputeDiffForBlock(block)
		elements = append(elements, element)
		current = compareActions(current, element.Action)
	})
	return elements, current
}

func (change Change) processMap(process func(key string, value Change)) {
	mapValue := change.asMap()

	handled := make(map[string]bool)
	for key := range mapValue.Before {
		handled[key] = true
		process(key, mapValue.getChild(key))
	}
	for key := range mapValue.After {
		if _, ok := handled[key]; ok {
			continue
		}
		process(key, mapValue.getChild(key))
	}
}
