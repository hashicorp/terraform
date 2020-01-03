package stdlib

import (
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var AbsoluteFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "num",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return args[0].Absolute(), nil
	},
})

var AddFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		// big.Float.Add can panic if the input values are opposing infinities,
		// so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't compute sum of opposing infinities")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()
		return args[0].Add(args[1]), nil
	},
})

var SubtractFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		// big.Float.Sub can panic if the input values are infinities,
		// so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't subtract infinity from itself")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()
		return args[0].Subtract(args[1]), nil
	},
})

var MultiplyFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		// big.Float.Mul can panic if the input values are both zero or both
		// infinity, so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't multiply zero by infinity")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()

		return args[0].Multiply(args[1]), nil
	},
})

var DivideFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		// big.Float.Quo can panic if the input values are both zero or both
		// infinity, so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't divide zero by zero or infinity by infinity")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()

		return args[0].Divide(args[1]), nil
	},
})

var ModuloFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		// big.Float.Mul can panic if the input values are both zero or both
		// infinity, so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = fmt.Errorf("can't use modulo with zero and infinity")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()

		return args[0].Modulo(args[1]), nil
	},
})

var GreaterThanFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].GreaterThan(args[1]), nil
	},
})

var GreaterThanOrEqualToFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].GreaterThanOrEqualTo(args[1]), nil
	},
})

var LessThanFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].LessThan(args[1]), nil
	},
})

var LessThanOrEqualToFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "a",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
		{
			Name:             "b",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].LessThanOrEqualTo(args[1]), nil
	},
})

var NegateFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "num",
			Type:             cty.Number,
			AllowDynamicType: true,
			AllowMarked:      true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return args[0].Negate(), nil
	},
})

var MinFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "numbers",
		Type:             cty.Number,
		AllowDynamicType: true,
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return cty.NilVal, fmt.Errorf("must pass at least one number")
		}

		min := cty.PositiveInfinity
		for _, num := range args {
			if num.LessThan(min).True() {
				min = num
			}
		}

		return min, nil
	},
})

var MaxFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	VarParam: &function.Parameter{
		Name:             "numbers",
		Type:             cty.Number,
		AllowDynamicType: true,
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return cty.NilVal, fmt.Errorf("must pass at least one number")
		}

		max := cty.NegativeInfinity
		for _, num := range args {
			if num.GreaterThan(max).True() {
				max = num
			}
		}

		return max, nil
	},
})

var IntFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "num",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		bf := args[0].AsBigFloat()
		if bf.IsInt() {
			return args[0], nil
		}
		bi, _ := bf.Int(nil)
		bf = (&big.Float{}).SetInt(bi)
		return cty.NumberVal(bf), nil
	},
})

// Absolute returns the magnitude of the given number, without its sign.
// That is, it turns negative values into positive values.
func Absolute(num cty.Value) (cty.Value, error) {
	return AbsoluteFunc.Call([]cty.Value{num})
}

// Add returns the sum of the two given numbers.
func Add(a cty.Value, b cty.Value) (cty.Value, error) {
	return AddFunc.Call([]cty.Value{a, b})
}

// Subtract returns the difference between the two given numbers.
func Subtract(a cty.Value, b cty.Value) (cty.Value, error) {
	return SubtractFunc.Call([]cty.Value{a, b})
}

// Multiply returns the product of the two given numbers.
func Multiply(a cty.Value, b cty.Value) (cty.Value, error) {
	return MultiplyFunc.Call([]cty.Value{a, b})
}

// Divide returns a divided by b, where both a and b are numbers.
func Divide(a cty.Value, b cty.Value) (cty.Value, error) {
	return DivideFunc.Call([]cty.Value{a, b})
}

// Negate returns the given number multipled by -1.
func Negate(num cty.Value) (cty.Value, error) {
	return NegateFunc.Call([]cty.Value{num})
}

// LessThan returns true if a is less than b.
func LessThan(a cty.Value, b cty.Value) (cty.Value, error) {
	return LessThanFunc.Call([]cty.Value{a, b})
}

// LessThanOrEqualTo returns true if a is less than b.
func LessThanOrEqualTo(a cty.Value, b cty.Value) (cty.Value, error) {
	return LessThanOrEqualToFunc.Call([]cty.Value{a, b})
}

// GreaterThan returns true if a is less than b.
func GreaterThan(a cty.Value, b cty.Value) (cty.Value, error) {
	return GreaterThanFunc.Call([]cty.Value{a, b})
}

// GreaterThanOrEqualTo returns true if a is less than b.
func GreaterThanOrEqualTo(a cty.Value, b cty.Value) (cty.Value, error) {
	return GreaterThanOrEqualToFunc.Call([]cty.Value{a, b})
}

// Modulo returns the remainder of a divided by b under integer division,
// where both a and b are numbers.
func Modulo(a cty.Value, b cty.Value) (cty.Value, error) {
	return ModuloFunc.Call([]cty.Value{a, b})
}

// Min returns the minimum number from the given numbers.
func Min(numbers ...cty.Value) (cty.Value, error) {
	return MinFunc.Call(numbers)
}

// Max returns the maximum number from the given numbers.
func Max(numbers ...cty.Value) (cty.Value, error) {
	return MaxFunc.Call(numbers)
}

// Int removes the fractional component of the given number returning an
// integer representing the whole number component, rounding towards zero.
// For example, -1.5 becomes -1.
//
// If an infinity is passed to Int, an error is returned.
func Int(num cty.Value) (cty.Value, error) {
	if num == cty.PositiveInfinity || num == cty.NegativeInfinity {
		return cty.NilVal, fmt.Errorf("can't truncate infinity to an integer")
	}
	return IntFunc.Call([]cty.Value{num})
}
