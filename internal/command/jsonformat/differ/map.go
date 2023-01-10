package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeAttributeChangeAsMap(elementType cty.Type) change.Change {
	mapValue := v.asMap()
	elements, current := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) (change.Change, plans.Action) {
		element := mapValue.getChild(key).computeChangeForType(elementType)
		return element, element.Action()
	})
	return change.New(change.Map(elements), current, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedMap(attributes map[string]*jsonprovider.Attribute) change.Change {
	mapValue := v.asMap()
	elements, current := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) (change.Change, plans.Action) {
		element := mapValue.getChild(key).computeChangeForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		return element, element.Action()
	})
	return change.New(change.Map(elements), current, v.replacePath())
}

func (v Value) computeBlockChangesAsMap(block *jsonprovider.Block) ([]change.Change, plans.Action) {
	mapValue := v.asMap()
	elements, action := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) (change.Change, plans.Action) {
		element := mapValue.getChild(key).ComputeChangeForBlock(block)
		return element, element.Action()
	})

	var ret []change.Change
	for _, element := range elements {
		ret = append(ret, element)
	}
	return ret, action
}

func (v Value) processMap(process func(key string, value Value)) {
	mapValue := v.asMap()

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
