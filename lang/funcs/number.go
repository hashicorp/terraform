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

// Ceil returns the closest whole number greater than or equal to the given value.
func Ceil(num cty.Value) (cty.Value, error) {
	return CeilFunc.Call([]cty.Value{num})
}
