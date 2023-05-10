// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func ComputeDiffForBlock(change structured.Change, block *jsonprovider.Block) computed.Diff {
	if sensitive, ok := checkForSensitiveBlock(change, block); ok {
		return sensitive
	}

	if unknown, ok := checkForUnknownBlock(change, block); ok {
		return unknown
	}

	current := change.GetDefaultActionForIteration()

	blockValue := change.AsMap()

	attributes := make(map[string]computed.Diff)
	for key, attr := range block.Attributes {
		childValue := blockValue.GetChild(key)

		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}

		// Empty strings in blocks should be considered null for legacy reasons.
		// The SDK doesn't support null strings yet, so we work around this now.
		if before, ok := childValue.Before.(string); ok && len(before) == 0 {
			childValue.Before = nil
		}
		if after, ok := childValue.After.(string); ok && len(after) == 0 {
			childValue.After = nil
		}

		// Always treat changes to blocks as implicit.
		childValue.BeforeExplicit = false
		childValue.AfterExplicit = false

		childChange := ComputeDiffForAttribute(childValue, attr)
		if childChange.Action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}

		attributes[key] = childChange
		current = collections.CompareActions(current, childChange.Action)
	}

	blocks := renderers.Blocks{
		ReplaceBlocks:         make(map[string]bool),
		BeforeSensitiveBlocks: make(map[string]bool),
		AfterSensitiveBlocks:  make(map[string]bool),
		SingleBlocks:          make(map[string]computed.Diff),
		ListBlocks:            make(map[string][]computed.Diff),
		SetBlocks:             make(map[string][]computed.Diff),
		MapBlocks:             make(map[string]map[string]computed.Diff),
	}

	for key, blockType := range block.BlockTypes {
		childValue := blockValue.GetChild(key)

		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}

		beforeSensitive := childValue.IsBeforeSensitive()
		afterSensitive := childValue.IsAfterSensitive()
		forcesReplacement := childValue.ReplacePaths.Matches()

		switch NestingMode(blockType.NestingMode) {
		case nestingModeSet:
			diffs, action := computeBlockDiffsAsSet(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllSetBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, action)
		case nestingModeList:
			diffs, action := computeBlockDiffsAsList(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllListBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, action)
		case nestingModeMap:
			diffs, action := computeBlockDiffsAsMap(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllMapBlocks(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, action)
		case nestingModeSingle, nestingModeGroup:
			diff := ComputeDiffForBlock(childValue, blockType.Block)
			if diff.Action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddSingleBlock(key, diff, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, diff.Action)
		default:
			panic("unrecognized nesting mode: " + blockType.NestingMode)
		}
	}

	return computed.NewDiff(renderers.Block(attributes, blocks), current, change.ReplacePaths.Matches())
}
