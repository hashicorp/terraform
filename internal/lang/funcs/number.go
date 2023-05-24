// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"math"
	"math/big"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

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
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
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
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
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
	Type:         function.StaticReturnType(cty.Number),
	RefineResult: refineNotNull,
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

// ParseIntFunc contructs a function that parses a string argument and returns an integer of the specified base.
var ParseIntFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:        "number",
			Type:        cty.DynamicPseudoType,
			AllowMarked: true,
		},
		{
			Name:        "base",
			Type:        cty.Number,
			AllowMarked: true,
		},
	},

	Type: func(args []cty.Value) (cty.Type, error) {
		if !args[0].Type().Equals(cty.String) {
			return cty.Number, function.NewArgErrorf(0, "first argument must be a string, not %s", args[0].Type().FriendlyName())
		}
		return cty.Number, nil
	},
	RefineResult: refineNotNull,

	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		var numstr string
		var base int
		var err error

		numArg, numMarks := args[0].Unmark()
		if err = gocty.FromCtyValue(numArg, &numstr); err != nil {
			return cty.UnknownVal(cty.String), function.NewArgError(0, err)
		}

		baseArg, baseMarks := args[1].Unmark()
		if err = gocty.FromCtyValue(baseArg, &base); err != nil {
			return cty.UnknownVal(cty.Number), function.NewArgError(1, err)
		}

		if base < 2 || base > 62 {
			return cty.UnknownVal(cty.Number), function.NewArgErrorf(
				1,
				"base must be a whole number between 2 and 62 inclusive",
			)
		}

		num, ok := (&big.Int{}).SetString(numstr, base)
		if !ok {
			return cty.UnknownVal(cty.Number), function.NewArgErrorf(
				0,
				"cannot parse %s as a base %s integer",
				redactIfSensitive(numstr, numMarks),
				redactIfSensitive(base, baseMarks),
			)
		}

		parsedNum := cty.NumberVal((&big.Float{}).SetInt(num)).WithMarks(numMarks, baseMarks)

		return parsedNum, nil
	},
})

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

// ParseInt parses a string argument and returns an integer of the specified base.
func ParseInt(num cty.Value, base cty.Value) (cty.Value, error) {
	return ParseIntFunc.Call([]cty.Value{num, base})
}
