package stdlib

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var NotFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "val",
			Type:             cty.Bool,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return args[0].Not(), nil
	},
})

var AndFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Bool,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Bool,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return args[0].And(args[1]), nil
	},
})

var OrFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Bool,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Bool,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return args[0].Or(args[1]), nil
	},
})

// Not returns the logical complement of the given boolean value.
func Not(num cty.Value) (cty.Value, error) {
	return NotFunc.Call([]cty.Value{num})
}

// And returns true if and only if both of the given boolean values are true.
func And(a, b cty.Value) (cty.Value, error) {
	return AndFunc.Call([]cty.Value{a, b})
}

// Or returns true if either of the given boolean values are true.
func Or(a, b cty.Value) (cty.Value, error) {
	return OrFunc.Call([]cty.Value{a, b})
}
