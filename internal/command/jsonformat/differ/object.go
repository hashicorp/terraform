// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func computeAttributeDiffAsObject(change structured.Change, attributes map[string]cty.Type) computed.Diff {
	attributeDiffs, action := processObject(change, attributes, func(value structured.Change, ctype cty.Type) computed.Diff {
		return ComputeDiffForType(value, ctype)
	})
	return computed.NewDiff(renderers.Object(attributeDiffs), action, change.ReplacePaths.Matches())
}

func computeAttributeDiffAsNestedObject(change structured.Change, attributes map[string]*jsonprovider.Attribute) computed.Diff {
	attributeDiffs, action := processObject(change, attributes, func(value structured.Change, attribute *jsonprovider.Attribute) computed.Diff {
		return ComputeDiffForAttribute(value, attribute)
	})
	return computed.NewDiff(renderers.NestedObject(attributeDiffs), action, change.ReplacePaths.Matches())
}

// processObject steps through the children of value as if it is an object and
// calls out to the provided computeDiff function once it has collated the
// diffs for each child attribute.
//
// We have to make this generic as attributes and nested objects process either
// cty.Type or jsonprovider.Attribute children respectively. And we want to
// reuse as much code as possible.
//
// Also, as it generic we cannot make this function a method on Change as you
// can't create generic methods on structs. Instead, we make this a generic
// function that receives the value as an argument.
func processObject[T any](v structured.Change, attributes map[string]T, computeDiff func(structured.Change, T) computed.Diff) (map[string]computed.Diff, plans.Action) {
	attributeDiffs := make(map[string]computed.Diff)
	mapValue := v.AsMap()

	currentAction := v.GetDefaultActionForIteration()
	for key, attribute := range attributes {
		attributeValue := mapValue.GetChild(key)

		if !attributeValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			attributeValue = attributeValue.AsNoOp()
		}

		// We always assume changes to object are implicit.
		attributeValue.BeforeExplicit = false
		attributeValue.AfterExplicit = false

		attributeDiff := computeDiff(attributeValue, attribute)
		if attributeDiff.Action == plans.NoOp && attributeValue.Before == nil && attributeValue.After == nil {
			// We skip attributes of objects that are null both before and
			// after. We don't even count these as unchanged attributes.
			continue
		}
		attributeDiffs[key] = attributeDiff
		currentAction = collections.CompareActions(currentAction, attributeDiff.Action)
	}

	return attributeDiffs, currentAction
}
