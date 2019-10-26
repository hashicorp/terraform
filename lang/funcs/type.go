package funcs

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var TypeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:      "variable",
			Type:      cty.DynamicPseudoType,
			AllowNull: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		varType := getType(args[0])
		return cty.StringVal(varType), nil
	},
})

// MakeIsTypeFunc constructs a "is..." function, like "isstring", which checks
// the type of a given variable is of the expected type.
// The given string checkType can be any type supported by the cty package
func MakeIsTypeFunc(checkType string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:      "variable",
				Type:      cty.DynamicPseudoType,
				AllowNull: true,
			},
		},
		Type: function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
			checked := getType(args[0]) == checkType
			return cty.BoolVal(checked), nil
		},
	})
}

func getType(arg cty.Value) string {
	var varType string

	// We have some special cases here, because var.Type().FriendlyName() does
	// not always give what we want.
	// In case the input var is null, it returns "any value"
	// For collection types it returns "list/map/set of strings" etc
	// We only want "list/map/set" instead.
	if arg.IsNull() {
		varType = "null"
	} else if arg.Type().IsListType() {
		varType = "list"
	} else if arg.Type().IsMapType() {
		varType = "map"
	} else if arg.Type().IsSetType() {
		varType = "set"
	} else {
		varType = arg.Type().FriendlyName()
	}

	return varType
}

// Type returns the type of a given var as string
func Type(t cty.Value) (cty.Value, error) {
	return TypeFunc.Call([]cty.Value{t})
}
