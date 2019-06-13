package yaml

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// YAMLDecodeFunc is a cty function for decoding arbitrary YAML source code
// into a cty Value, using the ImpliedType and Unmarshal methods of the
// Standard pre-defined converter.
var YAMLDecodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "src",
			Type: cty.String,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if !args[0].IsKnown() {
			return cty.DynamicPseudoType, nil
		}
		if args[0].IsNull() {
			return cty.NilType, function.NewArgErrorf(0, "YAML source code cannot be null")
		}
		return Standard.ImpliedType([]byte(args[0].AsString()))
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if retType == cty.DynamicPseudoType {
			return cty.DynamicVal, nil
		}
		return Standard.Unmarshal([]byte(args[0].AsString()), retType)
	},
})

// YAMLEncodeFunc is a cty function for encoding an arbitrary cty value
// into YAML.
var YAMLEncodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowNull:        true,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if !args[0].IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}
		raw, err := Standard.Marshal(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(string(raw)), nil
	},
})
