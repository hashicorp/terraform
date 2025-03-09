// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/plans"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func ComputeDiffForAttribute(change structured.Change, attribute *jsonprovider.Attribute) computed.Diff {
	if attribute.AttributeNestedType != nil {
		return computeDiffForNestedAttribute(change, attribute.AttributeNestedType)
	}

	return ComputeDiffForType(change, unmarshalAttribute(attribute))
}

func computeDiffForNestedAttribute(change structured.Change, nested *jsonprovider.NestedType) computed.Diff {
	if sensitive, ok := checkForSensitiveNestedAttribute(change, nested); ok {
		return sensitive
	}

	if computed, ok := checkForUnknownNestedAttribute(change, nested); ok {
		return computed
	}

	switch NestingMode(nested.NestingMode) {
	case nestingModeSingle, nestingModeGroup:
		return computeAttributeDiffAsNestedObject(change, nested.Attributes)
	case nestingModeMap:
		return computeAttributeDiffAsNestedMap(change, nested.Attributes)
	case nestingModeList:
		return computeAttributeDiffAsNestedList(change, nested.Attributes)
	case nestingModeSet:
		return computeAttributeDiffAsNestedSet(change, nested.Attributes)
	default:
		panic("unrecognized nesting mode: " + nested.NestingMode)
	}
}

func computeDiffForWriteOnlyAttribute(change structured.Change, blockAction plans.Action) computed.Diff {
	renderer := renderers.WriteOnly(change.IsBeforeSensitive() || change.IsAfterSensitive())
	replacePathMatches := change.ReplacePaths.Matches()
	// Write-only diffs should always copy the behavior of the block they are in, except for updates
	// since we don't want them to be always highlighted.
	if blockAction == plans.Update {
		return computed.NewDiff(renderer, plans.NoOp, replacePathMatches)
	}
	return computed.NewDiff(renderer, blockAction, replacePathMatches)

}

func ComputeDiffForType(change structured.Change, ctype cty.Type) computed.Diff {
	if !change.NonLegacySchema {
		// Empty strings in blocks should be considered null, because the legacy
		// SDK can't always differentiate between null and empty strings and may
		// return either.
		if before, ok := change.Before.(string); ok && len(before) == 0 {
			change.Before = nil
		}
		if after, ok := change.After.(string); ok && len(after) == 0 {
			change.After = nil
		}
	}

	if sensitive, ok := checkForSensitiveType(change, ctype); ok {
		return sensitive
	}

	if computed, ok := checkForUnknownType(change, ctype); ok {
		return computed
	}

	switch {
	case ctype == cty.NilType, ctype == cty.DynamicPseudoType:
		// Forward nil or dynamic types over to be processed as outputs.
		// There is nothing particularly special about the way outputs are
		// processed that make this unsafe, we could just as easily call this
		// function computeChangeForDynamicValues(), but external callers will
		// only be in this situation when processing outputs so this function
		// is named for their benefit.
		return ComputeDiffForOutput(change)
	case ctype.IsPrimitiveType():
		return computeAttributeDiffAsPrimitive(change, ctype)
	case ctype.IsObjectType():
		return computeAttributeDiffAsObject(change, ctype.AttributeTypes())
	case ctype.IsMapType():
		return computeAttributeDiffAsMap(change, ctype.ElementType())
	case ctype.IsListType():
		return computeAttributeDiffAsList(change, ctype.ElementType())
	case ctype.IsTupleType():
		return computeAttributeDiffAsTuple(change, ctype.TupleElementTypes())
	case ctype.IsSetType():
		return computeAttributeDiffAsSet(change, ctype.ElementType())
	default:
		panic("unrecognized type: " + ctype.FriendlyName())
	}
}

func unmarshalAttribute(attribute *jsonprovider.Attribute) cty.Type {
	ctyType, err := ctyjson.UnmarshalType(attribute.AttributeType)
	if err != nil {
		panic("could not unmarshal attribute type: " + err.Error())
	}
	return ctyType
}
