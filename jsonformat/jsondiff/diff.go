// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsondiff

import (
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/jsonformat/collections"
	"github.com/hashicorp/terraform/jsonformat/computed"
	"github.com/hashicorp/terraform/jsonformat/structured"
)

type TransformPrimitiveJson func(before, after interface{}, ctype cty.Type, action plans.Action) computed.Diff
type TransformObjectJson func(map[string]computed.Diff, plans.Action) computed.Diff
type TransformArrayJson func([]computed.Diff, plans.Action) computed.Diff
type TransformUnknownJson func(computed.Diff, plans.Action) computed.Diff
type TransformSensitiveJson func(computed.Diff, bool, bool, plans.Action) computed.Diff
type TransformTypeChangeJson func(before, after computed.Diff, action plans.Action) computed.Diff

// JsonOpts defines the external callback functions that callers should
// implement to process the supplied diffs.
type JsonOpts struct {
	Primitive  TransformPrimitiveJson
	Object     TransformObjectJson
	Array      TransformArrayJson
	Unknown    TransformUnknownJson
	Sensitive  TransformSensitiveJson
	TypeChange TransformTypeChangeJson
}

// Transform accepts a generic before and after value that is assumed to be JSON
// formatted and transforms it into a computed.Diff, using the callbacks
// supplied in the JsonOpts class.
func (opts JsonOpts) Transform(change structured.Change) computed.Diff {
	if sensitive, ok := opts.processSensitive(change); ok {
		return sensitive
	}

	if unknown, ok := opts.processUnknown(change); ok {
		return unknown
	}

	beforeType := GetType(change.Before)
	afterType := GetType(change.After)

	deleted := afterType == Null && !change.AfterExplicit
	created := beforeType == Null && !change.BeforeExplicit

	if beforeType == afterType || (created || deleted) {
		targetType := beforeType
		if targetType == Null {
			targetType = afterType
		}
		return opts.processUpdate(change, targetType)
	}

	b := opts.processUpdate(change.AsDelete(), beforeType)
	a := opts.processUpdate(change.AsCreate(), afterType)
	return opts.TypeChange(b, a, plans.Update)
}

func (opts JsonOpts) processUpdate(change structured.Change, jtype Type) computed.Diff {
	switch jtype {
	case Null:
		return opts.processPrimitive(change, cty.NilType)
	case Bool:
		return opts.processPrimitive(change, cty.Bool)
	case String:
		return opts.processPrimitive(change, cty.String)
	case Number:
		return opts.processPrimitive(change, cty.Number)
	case Object:
		return opts.processObject(change.AsMap())
	case Array:
		return opts.processArray(change.AsSlice())
	default:
		panic("unrecognized json type: " + jtype)
	}
}

func (opts JsonOpts) processPrimitive(change structured.Change, ctype cty.Type) computed.Diff {
	beforeMissing := change.Before == nil && !change.BeforeExplicit
	afterMissing := change.After == nil && !change.AfterExplicit

	var action plans.Action
	switch {
	case beforeMissing && !afterMissing:
		action = plans.Create
	case !beforeMissing && afterMissing:
		action = plans.Delete
	case reflect.DeepEqual(change.Before, change.After):
		action = plans.NoOp
	default:
		action = plans.Update
	}

	return opts.Primitive(change.Before, change.After, ctype, action)
}

func (opts JsonOpts) processArray(change structured.ChangeSlice) computed.Diff {
	processIndices := func(beforeIx, afterIx int) computed.Diff {
		// It's actually really difficult to render the diffs when some indices
		// within a list are relevant and others aren't. To make this simpler
		// we just treat all children of a relevant list as also relevant, so we
		// ignore the relevant attributes field.
		//
		// Interestingly the terraform plan builder also agrees with this, and
		// never sets relevant attributes beneath lists or sets. We're just
		// going to enforce this logic here as well. If the list is relevant
		// (decided elsewhere), then every element in the list is also relevant.
		return opts.Transform(change.GetChild(beforeIx, afterIx))
	}

	isObjType := func(value interface{}) bool {
		return GetType(value) == Object
	}

	return opts.Array(collections.TransformSlice(change.Before, change.After, processIndices, isObjType))
}

func (opts JsonOpts) processObject(change structured.ChangeMap) computed.Diff {
	return opts.Object(collections.TransformMap(change.Before, change.After, change.AllKeys(), func(key string) computed.Diff {
		child := change.GetChild(key)
		if !child.RelevantAttributes.MatchesPartial() {
			child = child.AsNoOp()
		}

		return opts.Transform(child)
	}))
}

func (opts JsonOpts) processUnknown(change structured.Change) (computed.Diff, bool) {
	return change.CheckForUnknown(
		false,
		func(current structured.Change) computed.Diff {
			return opts.Unknown(computed.Diff{}, plans.Create)
		}, func(current structured.Change, before structured.Change) computed.Diff {
			return opts.Unknown(opts.Transform(before), plans.Update)
		},
	)
}

func (opts JsonOpts) processSensitive(change structured.Change) (computed.Diff, bool) {
	return change.CheckForSensitive(opts.Transform, func(inner computed.Diff, beforeSensitive, afterSensitive bool, action plans.Action) computed.Diff {
		return opts.Sensitive(inner, beforeSensitive, afterSensitive, action)
	})
}
