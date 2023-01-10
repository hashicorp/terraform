package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeAttributeChangeAsList(elementType cty.Type) change.Change {
	sliceValue := v.asSlice()

	processIndices := func(beforeIx, afterIx int) (change.Change, plans.Action) {
		element := sliceValue.getChild(beforeIx, afterIx, false).computeChangeForType(elementType)
		return element, element.Action()
	}

	isObjType := func(_ interface{}) bool {
		return elementType.IsObjectType()
	}

	elements, current := collections.TransformSlice(sliceValue.Before, sliceValue.After, processIndices, isObjType)
	return change.New(change.List(elements), current, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedList(attributes map[string]*jsonprovider.Attribute) change.Change {
	var elements []change.Change
	current := v.getDefaultActionForIteration()
	v.processNestedList(func(value Value) {
		element := value.computeChangeForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action())
	})
	return change.New(change.NestedList(elements), current, v.replacePath())
}

func (v Value) computeBlockChangesAsList(block *jsonprovider.Block) ([]change.Change, plans.Action) {
	var elements []change.Change
	current := v.getDefaultActionForIteration()
	v.processNestedList(func(value Value) {
		element := value.ComputeChangeForBlock(block)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action())
	})
	return elements, current
}

func (v Value) processNestedList(process func(value Value)) {
	sliceValue := v.asSlice()
	for ix := 0; ix < len(sliceValue.Before) || ix < len(sliceValue.After); ix++ {
		process(sliceValue.getChild(ix, ix, false))
	}
}
