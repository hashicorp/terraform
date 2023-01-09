package differ

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/plans"
)

const (
	jsonNumber = "number"
	jsonObject = "object"
	jsonArray  = "array"
	jsonBool   = "bool"
	jsonString = "string"
	jsonNull   = "null"
)

func (v Value) ComputeChangeForOutput() change.Change {
	if sensitive, ok := v.checkForSensitive(); ok {
		return sensitive
	}

	if computed, ok := v.checkForComputedType(cty.DynamicPseudoType); ok {
		return computed
	}

	beforeType := getJsonType(v.Before)
	afterType := getJsonType(v.After)

	valueToAttribute := func(v Value, jsonType string) change.Change {
		var res change.Change

		switch jsonType {
		case jsonNull:
			res = v.computeAttributeChangeAsPrimitive(cty.NilType)
		case jsonBool:
			res = v.computeAttributeChangeAsPrimitive(cty.Bool)
		case jsonString:
			res = v.computeAttributeChangeAsPrimitive(cty.String)
		case jsonNumber:
			res = v.computeAttributeChangeAsPrimitive(cty.Number)
		case jsonObject:
			res = v.computeAttributeChangeAsMap(cty.DynamicPseudoType)
		case jsonArray:
			res = v.computeAttributeChangeAsList(cty.DynamicPseudoType)
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
		return valueToAttribute(v, targetType)
	}

	before := valueToAttribute(Value{
		Before:          v.Before,
		BeforeSensitive: v.BeforeSensitive,
	}, beforeType)

	after := valueToAttribute(Value{
		After:          v.After,
		AfterSensitive: v.AfterSensitive,
		Unknown:        v.Unknown,
	}, afterType)

	return change.New(change.TypeChange(before, after), plans.Update, false)
}

func getJsonType(json interface{}) string {
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
