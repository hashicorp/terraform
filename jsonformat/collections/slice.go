// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import (
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
	// If before and after are the same length and is not a reordering
	// we want to compare elements on an individual basis
	if len(before) == len(after) && !isReorder(before, after) {
		for ix := range before {
			process(ix, ix)
		}
		return
	}

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

// isReorder returns true if every item of before can be found in after
func isReorder[Input any](before, after []Input) bool {
	// To be a reorder the length needs to be the same
	if len(before) != len(after) {
		return false
	}

	for _, b := range before {
		hasMatch := false
		for _, a := range after {
			if reflect.DeepEqual(b, a) {
				// Match found, no need to search anymore
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	return true
}
