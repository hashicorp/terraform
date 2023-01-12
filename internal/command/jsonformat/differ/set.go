package differ

import (
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) computeAttributeDiffAsSet(elementType cty.Type) computed.Diff {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	change.processSet(func(value Change) {
		element := value.computeDiffForType(elementType)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.Set(elements), current, change.ReplacePaths.ForcesReplacement())
}

func (change Change) computeAttributeDiffAsNestedSet(attributes map[string]*jsonprovider.Attribute) computed.Diff {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	change.processSet(func(value Change) {
		element := value.computeDiffForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.NestedSet(elements), current, change.ReplacePaths.ForcesReplacement())
}

func (change Change) computeBlockDiffsAsSet(block *jsonprovider.Block) ([]computed.Diff, plans.Action) {
	var elements []computed.Diff
	current := change.getDefaultActionForIteration()
	change.processSet(func(value Change) {
		element := value.ComputeDiffForBlock(block)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return elements, current
}

func (change Change) processSet(process func(value Change)) {
	sliceValue := change.asSlice()

	foundInBefore := make(map[int]int)
	foundInAfter := make(map[int]int)

	// O(n^2) operation here to find matching pairs in the set, so we can make
	// the display look pretty. There might be a better way to do this, so look
	// here for potential optimisations.

	for ix := 0; ix < len(sliceValue.Before); ix++ {
		matched := false
		for jx := 0; jx < len(sliceValue.After); jx++ {
			if _, ok := foundInAfter[jx]; ok {
				// We've already found a match for this after value.
				continue
			}

			child := sliceValue.getChild(ix, jx)
			if reflect.DeepEqual(child.Before, child.After) && child.isBeforeSensitive() == child.isAfterSensitive() && !child.isUnknown() {
				matched = true
				foundInBefore[ix] = jx
				foundInAfter[jx] = ix
			}
		}

		if !matched {
			foundInBefore[ix] = -1
		}
	}

	// Now everything in before should be a key in foundInBefore and a value
	// in foundInAfter. If a key is mapped to -1 in foundInBefore it means it
	// does not have an equivalent in foundInAfter and so has been deleted.
	// Everything in foundInAfter has a matching value in foundInBefore, but
	// some values in after may not be in foundInAfter. This means these values
	// are newly created.

	for ix := 0; ix < len(sliceValue.Before); ix++ {
		if jx := foundInBefore[ix]; jx >= 0 {
			child := sliceValue.getChild(ix, jx)
			process(child)
			continue
		}
		child := sliceValue.getChild(ix, len(sliceValue.After))
		process(child)
	}

	for jx := 0; jx < len(sliceValue.After); jx++ {
		if _, ok := foundInAfter[jx]; ok {
			// Then this value was handled in the previous for loop.
			continue
		}
		child := sliceValue.getChild(len(sliceValue.Before), jx)
		process(child)
	}
}
