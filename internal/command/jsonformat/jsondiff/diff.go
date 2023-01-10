package jsondiff

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/collections"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

type TransformPrimitiveJson[Output any] func(before, after interface{}, ctype cty.Type, action plans.Action) Output
type TransformObjectJson[Output any] func(map[string]Output, plans.Action) Output
type TransformArrayJson[Output any] func([]Output, plans.Action) Output
type TransformTypeChangeJson[Output any] func(before, after Output, action plans.Action) Output

type JsonOpts[Output any] struct {
	Primitive TransformPrimitiveJson[Output]
	Object    TransformObjectJson[Output]
	Array     TransformArrayJson[Output]
}

func (opts JsonOpts[Output]) Transform(before, after interface{}) Output {
	beforeType := GetType(before)
	afterType := GetType(after)

}

func (opts JsonOpts[Output]) processUpdate(before, after interface{}, jtype Type) Output {
	switch jtype {
	case Null:
		return opts.processPrimitive(before, after, cty.NilType)
	case Bool:
		return opts.processPrimitive(before, after, cty.Bool)
	case String:
		return opts.processPrimitive(before, after, cty.String)
	case Number:
		return opts.processPrimitive(before, after, cty.Number)
	case Object:
	case Array:
	default:
		panic("unrecognized json type: " + jtype)
	}
}

func (opts JsonOpts[Output]) processPrimitive(before, after interface{}, ctype cty.Type) Output {
	var action plans.Action
	switch {
	case before == nil && after != nil:
		action = plans.Create
	case before != nil && after == nil:
		action = plans.Delete
	case reflect.DeepEqual(before, after):
		action = plans.NoOp
	default:
		action = plans.Update
	}

	return opts.Primitive(before, after, ctype, action)
}

func (opts JsonOpts[Output]) processObject(before, after []interface{}) Output {

	processIndices := func(before, after int) (Output, plans.Action) {
		var b, a interface{}

		if beforeIx >= 0 && beforeIx < len(before) {
			b = before[beforeIx]
		}

		if afterIx >= 0 && afterIx < len(after) {
			a = after[afterIx]
		}

		return opts.
	}

	isObjType := func(value interface{}) bool {
		return GetType(value) == Object
	}

	elements, action := collections.TransformSlice(before, after, processIndices, isObjType)
}
