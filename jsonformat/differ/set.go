// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/jsonformat/collections"
	"github.com/hashicorp/terraform/jsonformat/computed"
	"github.com/hashicorp/terraform/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/jsonformat/structured"
	"github.com/hashicorp/terraform/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/jsonprovider"
	"github.com/hashicorp/terraform/plans"
)

func computeAttributeDiffAsSet(change structured.Change, elementType cty.Type) computed.Diff {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processSet(change, func(value structured.Change) {
		element := ComputeDiffForType(value, elementType)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.Set(elements), current, change.ReplacePaths.Matches())
}

func computeAttributeDiffAsNestedSet(change structured.Change, attributes map[string]*jsonprovider.Attribute) computed.Diff {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processSet(change, func(value structured.Change) {
		element := computeDiffForNestedAttribute(value, &jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.NestedSet(elements), current, change.ReplacePaths.Matches())
}

func computeBlockDiffsAsSet(change structured.Change, block *jsonprovider.Block) ([]computed.Diff, plans.Action) {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processSet(change, func(value structured.Change) {
		element := ComputeDiffForBlock(value, block)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return elements, current
}

func processSet(change structured.Change, process func(value structured.Change)) {
	sliceValue := change.AsSlice()

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

			child := sliceValue.GetChild(ix, jx)
			if reflect.DeepEqual(child.Before, child.After) && child.IsBeforeSensitive() == child.IsAfterSensitive() && !child.IsUnknown() {
				matched = true
				foundInBefore[ix] = jx
				foundInAfter[jx] = ix
			}
		}

		if !matched {
			foundInBefore[ix] = -1
		}
	}

	clearRelevantStatus := func(change structured.Change) structured.Change {
		// It's actually really difficult to render the diffs when some indices
		// within a slice are relevant and others aren't. To make this simpler
		// we just treat all children of a relevant list or set as also
		// relevant.
		//
		// Interestingly the terraform plan builder also agrees with this, and
		// never sets relevant attributes beneath lists or sets. We're just
		// going to enforce this logic here as well. If the collection is
		// relevant (decided elsewhere), then every element in the collection is
		// also relevant. To be clear, in practice even if we didn't do the
		// following explicitly the effect would be the same. It's just nicer
		// for us to be clear about the behaviour we expect.
		//
		// What makes this difficult is the fact that the beforeIx and afterIx
		// can be different, and it's quite difficult to work out which one is
		// the relevant one. For nested lists, block lists, and tuples it's much
		// easier because we always process the same indices in the before and
		// after.
		change.RelevantAttributes = attribute_path.AlwaysMatcher()
		return change
	}

	// Now everything in before should be a key in foundInBefore and a value
	// in foundInAfter. If a key is mapped to -1 in foundInBefore it means it
	// does not have an equivalent in foundInAfter and so has been deleted.
	// Everything in foundInAfter has a matching value in foundInBefore, but
	// some values in after may not be in foundInAfter. This means these values
	// are newly created.

	for ix := 0; ix < len(sliceValue.Before); ix++ {
		if jx := foundInBefore[ix]; jx >= 0 {
			child := clearRelevantStatus(sliceValue.GetChild(ix, jx))
			process(child)
			continue
		}
		child := clearRelevantStatus(sliceValue.GetChild(ix, len(sliceValue.After)))
		process(child)
	}

	for jx := 0; jx < len(sliceValue.After); jx++ {
		if _, ok := foundInAfter[jx]; ok {
			// Then this value was handled in the previous for loop.
			continue
		}
		child := clearRelevantStatus(sliceValue.GetChild(len(sliceValue.Before), jx))
		process(child)
	}
}
