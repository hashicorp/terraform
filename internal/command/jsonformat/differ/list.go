package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) computeAttributeDiffAsList(elementType cty.Type) computed.Diff {
	sliceValue := change.asSlice()

	processIndices := func(beforeIx, afterIx int) computed.Diff {
		return sliceValue.getChild(beforeIx, afterIx, false).computeDiffForType(elementType)
	}

	isObjType := func(_ interface{}) bool {
		return elementType.IsObjectType()
	}

	elements, current := collections.TransformSlice(sliceValue.Before, sliceValue.After, processIndices, isObjType)
	return computed.NewDiff(renderers.List(elements), current, change.replacePath())
}

func (change Change) computeAttributeDiffAsNestedList(attributes map[string]*jsonprovider.Attribute) computed.Diff {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	change.processNestedList(func(value Change) {
		element := value.computeDiffForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.NestedList(elements), current, change.replacePath())
}

func (change Change) computeBlockDiffsAsList(block *jsonprovider.Block) ([]computed.Diff, plans.Action) {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	change.processNestedList(func(value Change) {
		element := value.ComputeDiffForBlock(block)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return elements, current
}

func (change Change) processNestedList(process func(value Change)) {
	sliceValue := change.asSlice()
	for ix := 0; ix < len(sliceValue.Before) || ix < len(sliceValue.After); ix++ {
		process(sliceValue.getChild(ix, ix, false))
	}
}
