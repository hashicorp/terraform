package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
)

func (change Change) computeAttributeDiffAsTuple(elementTypes []cty.Type) computed.Diff {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	sliceValue := change.asSlice()
	for ix, elementType := range elementTypes {
		childValue := sliceValue.getChild(ix, ix)
		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}
		element := childValue.ComputeDiffForType(elementType)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	}
	return computed.NewDiff(renderers.List(elements), current, change.ReplacePaths.Matches())
}
