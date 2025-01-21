package list

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TransposeFunc returns a function that transposes a list of lists.
// Given a list of lists [[a1, a2], [b1, b2]], it returns [[a1, b1], [a2, b2]].
func TransposeFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "list",
				Type:             cty.DynamicPseudoType,
				AllowDynamicType: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			listVal := args[0]
			if !listVal.Type().IsTupleType() && !listVal.Type().IsListType() {
				return cty.NilType, fmt.Errorf("argument must be a list or tuple")
			}

			// All inner elements must be lists/tuples
			elemType := listVal.Type().ElementType()
			if !elemType.IsTupleType() && !elemType.IsListType() {
				return cty.NilType, fmt.Errorf("all elements must be lists or tuples")
			}

			return cty.List(cty.DynamicPseudoType), nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			inputList := args[0]

			// Handle empty list
			if inputList.LengthInt() == 0 {
				return cty.ListValEmpty(cty.DynamicPseudoType), nil
			}

			// Find the maximum length of inner lists
			maxLen := 0
			for it := inputList.ElementIterator(); it.Next(); {
				_, v := it.Element()
				if l := v.LengthInt(); l > maxLen {
					maxLen = l
				}
			}

			// Create the transposed list
			outerLen := inputList.LengthInt()
			transposed := make([][]cty.Value, maxLen)
			for i := range transposed {
				transposed[i] = make([]cty.Value, outerLen)
			}

			// Fill the transposed list
			for i := 0; i < outerLen; i++ {
				innerList := inputList.Index(cty.NumberIntVal(int64(i)))
				for j := 0; j < maxLen; j++ {
					if j < innerList.LengthInt() {
						transposed[j][i] = innerList.Index(cty.NumberIntVal(int64(j)))
					} else {
						transposed[j][i] = cty.NullVal(cty.DynamicPseudoType)
					}
				}
			}

			// Convert to cty.Value
			result := make([]cty.Value, maxLen)
			for i, row := range transposed {
				result[i] = cty.TupleVal(row)
			}

			return cty.ListVal(result), nil
		},
	})
}
