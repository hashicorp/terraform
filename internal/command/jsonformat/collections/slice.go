package collections

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"reflect"
)

type ProcessIndices[Output any] func(before, after int) (Output, plans.Action)
type IsObjType[Input any] func(input Input) bool

func TransformSlice[Input, Output any](before, after []Input, process ProcessIndices[Output], isObjType IsObjType[Input]) ([]Output, plans.Action) {
	current := plans.NoOp
	if before != nil && after == nil {
		current = plans.Delete
	}
	if before == nil && after != nil {
		current = plans.Create
	}

	var elements []Output

	lcs := objchange.LongestCommonSubsequence(before, after, func(before, after Input) bool {
		return reflect.DeepEqual(before, after)
	})

	var beforeIx, afterIx, lcsIx int
	for beforeIx < len(before) || afterIx < len(after) || lcsIx < len(lcs) {
		// Step through all the before values until we hit the next item in the
		// longest common subsequence. We are going to just say that all of
		// these have been deleted.
		for beforeIx < len(before) && (lcsIx >= len(lcs) || !reflect.DeepEqual(before[beforeIx], lcs[lcsIx])) {
			isObjectDiff := isObjType(before[beforeIx]) && afterIx < len(after) && isObjType(after[afterIx]) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx]))
			if isObjectDiff {
				element, action := process(beforeIx, afterIx)
				elements = append(elements, element)
				current = CompareActions(current, action)

				beforeIx++
				afterIx++
				continue
			}

			element, action := process(beforeIx, len(after))
			elements = append(elements, element)
			current = CompareActions(current, action)
			beforeIx++
		}

		// Now, step through all the after values until hit the next item in the
		// LCS. We are going to say that all of these have been created.
		for afterIx < len(after) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx])) {
			element, action := process(len(before), afterIx)
			elements = append(elements, element)
			current = CompareActions(current, action)
			afterIx++
		}

		// Finally, add the item in common as unchanged.
		if lcsIx < len(lcs) {
			element, action := process(beforeIx, afterIx)
			elements = append(elements, element)
			current = CompareActions(current, action)
			beforeIx++
			afterIx++
			lcsIx++
		}
	}

	return elements, current
}
