package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func checkForUnknownType(change structured.Change, ctype cty.Type) (computed.Diff, bool) {
	return checkForUnknown(change, false, func(value structured.Change) computed.Diff {
		return ComputeDiffForType(value, ctype)
	})
}
func checkForUnknownNestedAttribute(change structured.Change, attribute *jsonprovider.NestedType) (computed.Diff, bool) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range attribute.Attributes {
		childUnknown[key] = true
	}

	return checkForUnknown(change, childUnknown, func(value structured.Change) computed.Diff {
		return computeDiffForNestedAttribute(value, attribute)
	})
}

func checkForUnknownBlock(change structured.Change, block *jsonprovider.Block) (computed.Diff, bool) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range block.Attributes {
		childUnknown[key] = true
	}

	return checkForUnknown(change, childUnknown, func(value structured.Change) computed.Diff {
		return ComputeDiffForBlock(value, block)
	})
}

func checkForUnknown(change structured.Change, childUnknown interface{}, computeDiff func(value structured.Change) computed.Diff) (computed.Diff, bool) {
	unknown := change.IsUnknown()

	if !unknown {
		return computed.Diff{}, false
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	change.AfterExplicit = true

	if change.Before == nil {
		return asDiff(change, renderers.Unknown(computed.Diff{})), true
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.

	beforeValue := structured.Change{
		Before:             change.Before,
		BeforeSensitive:    change.BeforeSensitive,
		Unknown:            childUnknown,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
	return asDiff(change, renderers.Unknown(computeDiff(beforeValue))), true
}
