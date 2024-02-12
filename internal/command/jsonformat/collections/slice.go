// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
)

type TransformIndices func(before, after int) computed.Diff
type ProcessIndices func(before, after int)
type IsObjType[Input any] func(input Input) bool

func TransformSlice[Input any](before, after []Input, process TransformIndices, isObjType IsObjType[Input]) ([]computed.Diff, plans.Action) {
	current := plans.NoOp
	if before != nil && after == nil {
		current = plans.Delete
	}
	if before == nil && after != nil {
		current = plans.Create
	}

	var elements []computed.Diff
	ProcessSlice(before, after, func(before, after int) {
		element := process(before, after)
		elements = append(elements, element)
		current = CompareActions(current, element.Action)
	}, isObjType)
	return elements, current
}

func ProcessSlice[Input any](before, after []Input, process ProcessIndices, isObjType IsObjType[Input]) {
	// TODO: Put into a function
	allValuesPrimitive := true

	for _, item := range before {
		switch any(item).(type) {
		case int, string, bool:
			continue
		default:
			allValuesPrimitive = false
			break
		}
	}

	// If we are dealing with primitives we just diff on an individual level
	// TODO: Handle additions deletions
	// TODO: Add tests
	if allValuesPrimitive {
		for i := range before {
			process(i, i)
		}
		return
	}

	lcs := objchange.LongestCommonSubsequence(before, after, func(before, after Input) bool {
		// strings are not necessarily deep equal, but we want to handle the diffing on a
		// per item level

		_, beforeString := any(before).(string)
		_, afterString := any(after).(string)
		if beforeString && afterString {
			fmt.Println("Hey now")
			return false
		}

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
				process(beforeIx, afterIx)
				beforeIx++
				afterIx++
				continue
			}

			process(beforeIx, len(after))
			beforeIx++
		}

		// Now, step through all the after values until hit the next item in the
		// LCS. We are going to say that all of these have been created.
		for afterIx < len(after) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx])) {
			process(len(before), afterIx)
			afterIx++
		}

		// Finally, add the item in common as unchanged.
		if lcsIx < len(lcs) {
			process(beforeIx, afterIx)
			beforeIx++
			afterIx++
			lcsIx++
		}
	}
}
