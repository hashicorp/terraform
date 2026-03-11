// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

	// NonLegacyValue is only ever switched from false to true, since the
	// behavior would be for the entire resource.
	change.NonLegacySchema = change.NonLegacySchema || containsNonLegacyFeatures(block)

	current := change.GetDefaultActionForIteration()

	blockValue := change.AsMap()

	attributes := make(map[string]computed.Diff)
	for key, attr := range block.Attributes {
		if attr.WriteOnly {
			continue
		}

		childValue := blockValue.GetChild(key)

		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}

		// Always treat changes to blocks as implicit.
		childValue.BeforeExplicit = false
		childValue.AfterExplicit = false

		childChange := ComputeDiffForAttribute(childValue, attr)
		if childChange.Action == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks except if they are write-only.
			continue
		}

		attributes[key] = childChange
		current = collections.CompareActions(current, childChange.Action)
	}

	blocks := renderers.Blocks{
		ReplaceBlocks:         make(map[string]bool),
		BeforeSensitiveBlocks: make(map[string]bool),
		AfterSensitiveBlocks:  make(map[string]bool),
		UnknownBlocks:         make(map[string]bool),
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
		unknown := childValue.IsUnknown()

		switch NestingMode(blockType.NestingMode) {
		case nestingModeSet:
			diffs, action := computeBlockDiffsAsSet(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil && !unknown {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllSetBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive, unknown)
			current = collections.CompareActions(current, action)
		case nestingModeList:
			diffs, action := computeBlockDiffsAsList(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil && !unknown {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllListBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive, unknown)
			current = collections.CompareActions(current, action)
		case nestingModeMap:
			diffs, action := computeBlockDiffsAsMap(childValue, blockType.Block)
			if action == plans.NoOp && childValue.Before == nil && childValue.After == nil && !unknown {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllMapBlocks(key, diffs, forcesReplacement, beforeSensitive, afterSensitive, unknown)
			current = collections.CompareActions(current, action)
		case nestingModeSingle, nestingModeGroup:
			diff := ComputeDiffForBlock(childValue, blockType.Block)
			if diff.Action == plans.NoOp && childValue.Before == nil && childValue.After == nil && !unknown {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddSingleBlock(key, diff, forcesReplacement, beforeSensitive, afterSensitive, unknown)
			current = collections.CompareActions(current, diff.Action)
		default:
			panic("unrecognized nesting mode: " + blockType.NestingMode)
		}
	}

	for name, attr := range block.Attributes {
		if attr.WriteOnly {
			attributes[name] = computeDiffForWriteOnlyAttribute(change, current)
		}
	}

	return computed.NewDiff(renderers.Block(attributes, blocks), current, change.ReplacePaths.Matches())
}

// containsNonLegacyFeatures checks for features not supported by the legacy
// SDK, so that we can skip the empty string -> null fixup for them.
func containsNonLegacyFeatures(block *jsonprovider.Block) bool {
	for _, blockType := range block.BlockTypes {
		switch NestingMode(blockType.NestingMode) {
		case nestingModeMap, nestingModeGroup:
			// these block types were not possible in the SDK
			return true
		}
	}

	for _, attribute := range block.Attributes {
		//nested object types were not possible in the SDK
		if attribute.AttributeNestedType != nil {
			return true
		}

		ty := unmarshalAttribute(attribute)
		// these types were not possible in the SDK
		switch {
		case ty.HasDynamicTypes():
			return true
		case ty.IsTupleType() || ty.IsObjectType():
			return true
		case ty.IsCollectionType():
			// Nested collections were not really supported, but could be
			// generated with string types (though we conservatively limit this
			// to primitive types)
			ety := ty.ElementType()
			if ety.IsCollectionType() && !ety.ElementType().IsPrimitiveType() {
				return true
			}
		}
	}
	return false
}
