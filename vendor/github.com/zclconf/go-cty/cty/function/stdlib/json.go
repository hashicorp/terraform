package stdlib

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/json"
)

var JSONEncodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "val",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		val := args[0]
		if !val.IsWhollyKnown() {
			// We can't serialize unknowns, so if the value is unknown or
			// contains any _nested_ unknowns then our result must be
			// unknown.
			return cty.UnknownVal(retType), nil
		}

		buf, err := json.Marshal(val, val.Type())
		if err != nil {
			return cty.NilVal, err
		}

		return cty.StringVal(string(buf)), nil
	},
})

var JSONDecodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		str := args[0]
		if !str.IsKnown() {
			return cty.DynamicPseudoType, nil
		}

		buf := []byte(str.AsString())
		return json.ImpliedType(buf)
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		buf := []byte(args[0].AsString())
		return json.Unmarshal(buf, retType)
	},
})

// JSONEncode returns a JSON serialization of the given value.
func JSONEncode(val cty.Value) (cty.Value, error) {
	return JSONEncodeFunc.Call([]cty.Value{val})
}

// JSONDecode parses the given JSON string and, if it is valid, returns the
// value it represents.
//
// Note that applying JSONDecode to the result of JSONEncode may not produce
// an identically-typed result, since JSON encoding is lossy for cty Types.
// The resulting value will consist only of primitive types, object types, and
// tuple types.
func JSONDecode(str cty.Value) (cty.Value, error) {
	return JSONDecodeFunc.Call([]cty.Value{str})
}
