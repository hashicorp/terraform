// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/jsonformat/collections"
	"github.com/hashicorp/terraform/jsonformat/computed"
	"github.com/hashicorp/terraform/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/jsonformat/structured"
	"github.com/hashicorp/terraform/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/jsonprovider"
	"github.com/hashicorp/terraform/plans"
)

func computeAttributeDiffAsList(change structured.Change, elementType cty.Type) computed.Diff {
	sliceValue := change.AsSlice()

	processIndices := func(beforeIx, afterIx int) computed.Diff {
		value := sliceValue.GetChild(beforeIx, afterIx)

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
		value.RelevantAttributes = attribute_path.AlwaysMatcher()

		return ComputeDiffForType(value, elementType)
	}

	isObjType := func(_ interface{}) bool {
		return elementType.IsObjectType()
	}

	elements, current := collections.TransformSlice(sliceValue.Before, sliceValue.After, processIndices, isObjType)
	return computed.NewDiff(renderers.List(elements), current, change.ReplacePaths.Matches())
}

func computeAttributeDiffAsNestedList(change structured.Change, attributes map[string]*jsonprovider.Attribute) computed.Diff {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processNestedList(change, func(value structured.Change) {
		element := computeDiffForNestedAttribute(value, &jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return computed.NewDiff(renderers.NestedList(elements), current, change.ReplacePaths.Matches())
}

func computeBlockDiffsAsList(change structured.Change, block *jsonprovider.Block) ([]computed.Diff, plans.Action) {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processNestedList(change, func(value structured.Change) {
		element := ComputeDiffForBlock(value, block)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	})
	return elements, current
}

func processNestedList(change structured.Change, process func(value structured.Change)) {
	sliceValue := change.AsSlice()
	for ix := 0; ix < len(sliceValue.Before) || ix < len(sliceValue.After); ix++ {
		value := sliceValue.GetChild(ix, ix)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		process(value)
	}
}
