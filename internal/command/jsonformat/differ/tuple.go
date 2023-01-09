package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
)

func (v Value) computeAttributeChangeAsTuple(elementTypes []cty.Type) change.Change {
	var elements []change.Change
	current := v.getDefaultActionForIteration()
	sliceValue := v.asSlice()
	for ix, elementType := range elementTypes {
		childValue := sliceValue.getChild(ix, ix, false)
		element := childValue.computeChangeForType(elementType)
		elements = append(elements, element)
		current = compareActions(current, element.Action())
	}
	return change.New(change.List(elements), current, v.replacePath())
}
