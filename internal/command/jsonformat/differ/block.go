package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) ComputeDiffForBlock(block *jsonprovider.Block) computed.Diff {
	if sensitive, ok := change.checkForSensitiveBlock(block); ok {
		return sensitive
	}

	if computed, ok := change.checkForUnknownBlock(block); ok {
		return computed
	}

	current := change.getDefaultActionForIteration()

	blockValue := change.asMap()

	attributes := make(map[string]computed.Diff)
	for key, attr := range block.Attributes {
		childValue := blockValue.getChild(key)
		childChange := childValue.ComputeDiffForAttribute(attr)
		if childChange.Action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}

		attributes[key] = childChange
		current = compareActions(current, childChange.Action)
	}

	blocks := make(map[string][]computed.Diff)
	for key, blockType := range block.BlockTypes {
		childValue := blockValue.getChild(key)
		childChanges, next := childValue.computeDiffsForBlockType(blockType)
		if next == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}
		blocks[key] = childChanges
		current = compareActions(current, next)
	}

	return computed.NewDiff(renderers.Block(attributes, blocks), current, change.replacePath())
}

func (change Change) computeDiffsForBlockType(blockType *jsonprovider.BlockType) ([]computed.Diff, plans.Action) {
	switch NestingMode(blockType.NestingMode) {
	case nestingModeSet:
		return change.computeBlockDiffsAsSet(blockType.Block)
	case nestingModeList:
		return change.computeBlockDiffsAsList(blockType.Block)
	case nestingModeMap:
		return change.computeBlockDiffsAsMap(blockType.Block)
	case nestingModeSingle, nestingModeGroup:
		diff := change.ComputeDiffForBlock(blockType.Block)
		return []computed.Diff{diff}, diff.Action
	default:
		panic("unrecognized nesting mode: " + blockType.NestingMode)
	}
}
