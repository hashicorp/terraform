package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) computeAttributeDiffAsMap(elementType cty.Type) computed.Diff {
	mapValue := change.asMap()
	elements, current := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) computed.Diff {
		value := mapValue.getChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return value.ComputeDiffForType(elementType)
	})
	return computed.NewDiff(renderers.Map(elements), current, change.ReplacePaths.Matches())
}

func (change Change) computeAttributeDiffAsNestedMap(attributes map[string]*jsonprovider.Attribute) computed.Diff {
	mapValue := change.asMap()
	elements, current := collections.TransformMap(mapValue.Before, mapValue.After, func(key string) computed.Diff {
		value := mapValue.getChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return value.computeDiffForNestedAttribute(&jsonprovider.NestedType{
			Attributes:  attributes,
			NestingMode: "single",
		})
	})
	return computed.NewDiff(renderers.NestedMap(elements), current, change.ReplacePaths.Matches())
}

func (change Change) computeBlockDiffsAsMap(block *jsonprovider.Block) (map[string]computed.Diff, plans.Action) {
	mapValue := change.asMap()
	return collections.TransformMap(mapValue.Before, mapValue.After, func(key string) computed.Diff {
		value := mapValue.getChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return value.ComputeDiffForBlock(block)
	})
}
