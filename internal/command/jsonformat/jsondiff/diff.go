package jsondiff

import (
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ/attribute_path"
	"github.com/hashicorp/terraform/internal/plans"
)

type TransformPrimitiveJson func(before, after interface{}, ctype cty.Type, action plans.Action) computed.Diff
type TransformObjectJson func(map[string]computed.Diff, plans.Action) computed.Diff
type TransformArrayJson func([]computed.Diff, plans.Action) computed.Diff
type TransformTypeChangeJson func(before, after computed.Diff, action plans.Action) computed.Diff

// JsonOpts defines the external callback functions that callers should
// implement to process the supplied diffs.
type JsonOpts struct {
	Primitive  TransformPrimitiveJson
	Object     TransformObjectJson
	Array      TransformArrayJson
	TypeChange TransformTypeChangeJson
}

// Transform accepts a generic before and after value that is assumed to be JSON
// formatted and transforms it into a computed.Diff, using the callbacks
// supplied in the JsonOpts class.
func (opts JsonOpts) Transform(before, after interface{}, beforeExplicit, afterExplicit bool, relevantAttributes attribute_path.Matcher) computed.Diff {
	beforeType := GetType(before)
	afterType := GetType(after)

	deleted := afterType == Null && !afterExplicit
	created := beforeType == Null && !beforeExplicit

	if beforeType == afterType || (created || deleted) {
		targetType := beforeType
		if targetType == Null {
			targetType = afterType
		}
		return opts.processUpdate(before, after, beforeExplicit, afterExplicit, targetType, relevantAttributes)
	}

	b := opts.processUpdate(before, nil, beforeExplicit, false, beforeType, relevantAttributes)
	a := opts.processUpdate(nil, after, false, afterExplicit, afterType, relevantAttributes)
	return opts.TypeChange(b, a, plans.Update)
}

func (opts JsonOpts) processUpdate(before, after interface{}, beforeExplicit, afterExplicit bool, jtype Type, relevantAttributes attribute_path.Matcher) computed.Diff {
	switch jtype {
	case Null:
		return opts.processPrimitive(before, after, beforeExplicit, afterExplicit, cty.NilType)
	case Bool:
		return opts.processPrimitive(before, after, beforeExplicit, afterExplicit, cty.Bool)
	case String:
		return opts.processPrimitive(before, after, beforeExplicit, afterExplicit, cty.String)
	case Number:
		return opts.processPrimitive(before, after, beforeExplicit, afterExplicit, cty.Number)
	case Object:
		var b, a map[string]interface{}

		if before != nil {
			b = before.(map[string]interface{})
		}

		if after != nil {
			a = after.(map[string]interface{})
		}

		return opts.processObject(b, a, relevantAttributes)
	case Array:
		var b, a []interface{}

		if before != nil {
			b = before.([]interface{})
		}

		if after != nil {
			a = after.([]interface{})
		}

		return opts.processArray(b, a)
	default:
		panic("unrecognized json type: " + jtype)
	}
}

func (opts JsonOpts) processPrimitive(before, after interface{}, beforeExplicit, afterExplicit bool, ctype cty.Type) computed.Diff {
	beforeMissing := before == nil && !beforeExplicit
	afterMissing := after == nil && !afterExplicit

	var action plans.Action
	switch {
	case beforeMissing && !afterMissing:
		action = plans.Create
	case !beforeMissing && afterMissing:
		action = plans.Delete
	case reflect.DeepEqual(before, after):
		action = plans.NoOp
	default:
		action = plans.Update
	}

	return opts.Primitive(before, after, ctype, action)
}

func (opts JsonOpts) processArray(before, after []interface{}) computed.Diff {
	processIndices := func(beforeIx, afterIx int) computed.Diff {
		var b, a interface{}

		beforeExplicit := false
		afterExplicit := false

		if beforeIx >= 0 && beforeIx < len(before) {
			b = before[beforeIx]
			beforeExplicit = true
		}
		if afterIx >= 0 && afterIx < len(after) {
			a = after[afterIx]
			afterExplicit = true
		}

		// It's actually really difficult to render the diffs when some indices
		// within a list are relevant and others aren't. To make this simpler
		// we just treat all children of a relevant list as also relevant.
		//
		// Interestingly the terraform plan builder also agrees with this, and
		// never sets relevant attributes beneath lists or sets. We're just
		// going to enforce this logic here as well. If the list is relevant
		// (decided elsewhere), then every element in the list is also relevant.
		return opts.Transform(b, a, beforeExplicit, afterExplicit, attribute_path.AlwaysMatcher())
	}

	isObjType := func(value interface{}) bool {
		return GetType(value) == Object
	}

	return opts.Array(collections.TransformSlice(before, after, processIndices, isObjType))
}

func (opts JsonOpts) processObject(before, after map[string]interface{}, relevantAttributes attribute_path.Matcher) computed.Diff {
	return opts.Object(collections.TransformMap(before, after, func(key string) computed.Diff {
		beforeChild, beforeExplicit := before[key]
		afterChild, afterExplicit := after[key]

		childRelevantAttributes := relevantAttributes.GetChildWithKey(key)
		if !childRelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			afterChild = beforeChild
			afterExplicit = beforeExplicit

		}

		return opts.Transform(beforeChild, afterChild, beforeExplicit, afterExplicit, childRelevantAttributes)
	}))
}
