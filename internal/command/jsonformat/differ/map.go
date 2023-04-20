package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func computeAttributeDiffAsMap(change structured.Change, elementType cty.Type) computed.Diff {
	mapValue := change.AsMap()

	// The jsonplan package will have stripped out unknowns from our after value
	// so we're going to add them back in here.
	//
	// This only affects attributes and not nested attributes or blocks, so we
	// only perform this fix in this function and not the equivalent map
	// functions for nested attributes and blocks.

	// There is actually a difference between a null map and an empty map for
	// purposes of calculating a delete, create, or update operation.

	var after map[string]interface{}
	if mapValue.After != nil {
		after = make(map[string]interface{})
	}

	for key, value := range mapValue.After {
		after[key] = value
	}
	for key := range mapValue.Unknown {
		if _, ok := after[key]; ok {
			// Then this unknown value was in after, this probably means it has
			// a child that is unknown rather than being unknown itself. As
			// such, we'll skip over it. Note, it doesn't particularly matter if
			// an element is in both places - it's just important we actually
			// do cover all the elements. We want a complete union and therefore
			// duplicates are no cause for concern as long as we dedupe here.
			continue
		}
		after[key] = nil
	}

	elements, current := collections.TransformMap(mapValue.Before, after, func(key string) computed.Diff {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return ComputeDiffForType(value, elementType)
	})
	return computed.NewDiff(renderers.Map(elements), current, change.ReplacePaths.Matches())
}

func computeAttributeDiffAsNestedMap(change structured.Change, attributes map[string]*jsonprovider.Attribute) computed.Diff {
	mapValue := change.AsMap()
	elements, current := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) computed.Diff {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return computeDiffForNestedAttribute(value, &jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
	})
	return computed.NewDiff(renderers.NestedMap(elements), current, change.ReplacePaths.Matches())
}

func computeBlockDiffsAsMap(change structured.Change, block *jsonprovider.Block) (map[string]computed.Diff, plans.Action) {
	mapValue := change.AsMap()
	return collections.TransformMap(mapValue.Before, mapValue.After, func(key string) computed.Diff {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return ComputeDiffForBlock(value, block)
	})
}
