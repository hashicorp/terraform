// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/jsonformat/collections"
	"github.com/hashicorp/terraform/jsonformat/computed"
	"github.com/hashicorp/terraform/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/jsonformat/structured"
)

func computeAttributeDiffAsTuple(change structured.Change, elementTypes []cty.Type) computed.Diff {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	sliceValue := change.AsSlice()
	for ix, elementType := range elementTypes {
		childValue := sliceValue.GetChild(ix, ix)
		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}
		element := ComputeDiffForType(childValue, elementType)
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	}
	return computed.NewDiff(renderers.List(elements), current, change.ReplacePaths.Matches())
}
