package differ

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

func (change Change) ComputeDiffForOutput() computed.Diff {
	if sensitive, ok := change.checkForSensitiveType(cty.DynamicPseudoType); ok {
		return sensitive
	}

	if unknown, ok := change.checkForUnknownType(cty.DynamicPseudoType); ok {
		return unknown
	}

	beforeType := getJsonType(change.Before)
	afterType := getJsonType(change.After)

	valueToAttribute := func(v Change, jsonType JsonType) computed.Diff {
		var res computed.Diff

		switch jsonType {
		case jsonNull:
			res = v.computeAttributeDiffAsPrimitive(cty.NilType)
		case jsonBool:
			res = v.computeAttributeDiffAsPrimitive(cty.Bool)
		case jsonString:
			res = v.computeAttributeDiffAsPrimitive(cty.String)
		case jsonNumber:
			res = v.computeAttributeDiffAsPrimitive(cty.Number)
		case jsonObject:
			res = v.computeAttributeDiffAsMap(cty.DynamicPseudoType)
		case jsonArray:
			res = v.computeAttributeDiffAsList(cty.DynamicPseudoType)
		default:
			panic("unrecognized json type: " + jsonType)
		}

		return res
	}

	if beforeType == afterType || (beforeType == jsonNull || afterType == jsonNull) {
		targetType := beforeType
		if targetType == jsonNull {
			targetType = afterType
		}
		return valueToAttribute(change, targetType)
	}

	before := valueToAttribute(Change{
		Before:          change.Before,
		BeforeSensitive: change.BeforeSensitive,
	}, beforeType)

	after := valueToAttribute(Change{
		After:          change.After,
		AfterSensitive: change.AfterSensitive,
		Unknown:        change.Unknown,
	}, afterType)

	return computed.NewDiff(renderers.TypeChange(before, after), plans.Update, false)
}

func getJsonType(json interface{}) JsonType {
	switch json.(type) {
	case []interface{}:
		return jsonArray
	case float64:
		return jsonNumber
	case string:
		return jsonString
	case bool:
		return jsonBool
	case nil:
		return jsonNull
	case map[string]interface{}:
		return jsonObject
	default:
		panic(fmt.Sprintf("unrecognized json type %T", json))
	}
}
