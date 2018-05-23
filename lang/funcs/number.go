package funcs

import (
	"math"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

// CeilFunc contructs a function that returns the closest whole number greater
// than or equal to the given value.
var CeilFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "num",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var val float64
		if err := gocty.FromCtyValue(args[0], &val); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		return cty.NumberIntVal(int64(math.Ceil(val))), nil
	},
})

// FloorFunc contructs a function that returns the closest whole number lesser
// than or equal to the given value.
var FloorFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "num",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var val float64
		if err := gocty.FromCtyValue(args[0], &val); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		return cty.NumberIntVal(int64(math.Floor(val))), nil
	},
})

// LogFunc contructs a function that returns the logarithm of a given number in a given base.
var LogFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "num",
			Type: cty.Number,
		},
		{
			Name: "base",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var num float64
		if err := gocty.FromCtyValue(args[0], &num); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		var base float64
		if err := gocty.FromCtyValue(args[1], &base); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.NumberFloatVal(math.Log(num) / math.Log(base)), nil
	},
})

// PowFunc contructs a function that returns the logarithm of a given number in a given base.
var PowFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "num",
			Type: cty.Number,
		},
		{
			Name: "power",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var num float64
		if err := gocty.FromCtyValue(args[0], &num); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		var power float64
		if err := gocty.FromCtyValue(args[1], &power); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.NumberFloatVal(math.Pow(num, power)), nil
	},
})

// SignumFunc contructs a function that returns the closest whole number greater
// than or equal to the given value.
var SignumFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "num",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var num int
		if err := gocty.FromCtyValue(args[0], &num); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		switch {
		case num < 0:
			return cty.NumberIntVal(-1), nil
		case num > 0:
			return cty.NumberIntVal(+1), nil
		default:
			return cty.NumberIntVal(0), nil
		}
	},
})

// Ceil returns the closest whole number greater than or equal to the given value.
func Ceil(num cty.Value) (cty.Value, error) {
	return CeilFunc.Call([]cty.Value{num})
}

// Floor returns the closest whole number lesser than or equal to the given value.
func Floor(num cty.Value) (cty.Value, error) {
	return FloorFunc.Call([]cty.Value{num})
}

// Log returns returns the logarithm of a given number in a given base.
func Log(num, base cty.Value) (cty.Value, error) {
	return LogFunc.Call([]cty.Value{num, base})
}

// Pow returns the logarithm of a given number in a given base.
func Pow(num, power cty.Value) (cty.Value, error) {
	return PowFunc.Call([]cty.Value{num, power})
}

// Signum determines the sign of a number, returning a number between -1 and
// 1 to represent the sign.
func Signum(num cty.Value) (cty.Value, error) {
	return SignumFunc.Call([]cty.Value{num})
}
