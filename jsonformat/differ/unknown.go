// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/jsonformat/computed"
	"github.com/hashicorp/terraform/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/jsonformat/structured"
	"github.com/hashicorp/terraform/jsonprovider"
)

func checkForUnknownType(change structured.Change, ctype cty.Type) (computed.Diff, bool) {
	return change.CheckForUnknown(
		false,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) computed.Diff {
			return ComputeDiffForType(value, ctype)
		}))
}

func checkForUnknownNestedAttribute(change structured.Change, attribute *jsonprovider.NestedType) (computed.Diff, bool) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range attribute.Attributes {
		childUnknown[key] = true
	}

	return change.CheckForUnknown(
		childUnknown,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) computed.Diff {
			return computeDiffForNestedAttribute(value, attribute)
		}))
}

func checkForUnknownBlock(change structured.Change, block *jsonprovider.Block) (computed.Diff, bool) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range block.Attributes {
		childUnknown[key] = true
	}

	return change.CheckForUnknown(
		childUnknown,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) computed.Diff {
			return ComputeDiffForBlock(value, block)
		}))
}

func processUnknown(current structured.Change) computed.Diff {
	return asDiff(current, renderers.Unknown(computed.Diff{}))
}

func createProcessUnknownWithBefore(computeDiff func(value structured.Change) computed.Diff) structured.ProcessUnknownWithBefore {
	return func(current structured.Change, before structured.Change) computed.Diff {
		return asDiff(current, renderers.Unknown(computeDiff(before)))
	}
}
