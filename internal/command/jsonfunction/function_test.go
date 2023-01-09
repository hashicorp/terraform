package jsonfunction

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestMarshalFunction(t *testing.T) {
	tests := []struct {
		Name  string
		Input function.Function
		Want  *FunctionSignature
	}{
		{
			"minimal function",
			function.New(&function.Spec{
				Type: function.StaticReturnType(cty.Bool),
			}),
			&FunctionSignature{
				ReturnType: "bool",
			},
		},
		{
			"function with description",
			function.New(&function.Spec{
				Description: "`timestamp` returns a UTC timestamp string.",
				Type:        function.StaticReturnType(cty.String),
			}),
			&FunctionSignature{
				Description: "`timestamp` returns a UTC timestamp string.",
				ReturnType:  "string",
			},
		},
		{
			"function with parameters",
			function.New(&function.Spec{
				Params: []function.Parameter{
					{
						Name:        "timestamp",
						Description: "timestamp text",
						Type:        cty.String,
					},
					{
						Name:        "duration",
						Description: "duration text",
						Type:        cty.String,
					},
				},
				Type: function.StaticReturnType(cty.String),
			}),
			&FunctionSignature{
				ReturnType: "string",
				Parameters: []*parameter{
					{
						Name:        "timestamp",
						Description: "timestamp text",
						Type:        "string",
					},
					{
						Name:        "duration",
						Description: "duration text",
						Type:        "string",
					},
				},
			},
		},
		{
			"function with variadic parameter",
			function.New(&function.Spec{
				VarParam: &function.Parameter{
					Name:             "default",
					Description:      "default description",
					Type:             cty.DynamicPseudoType,
					AllowUnknown:     true,
					AllowDynamicType: true,
					AllowNull:        true,
					AllowMarked:      true,
				},
				Type: function.StaticReturnType(cty.DynamicPseudoType),
			}),
			&FunctionSignature{
				ReturnType: "dynamic",
				VariadicParameter: &parameter{
					Name:        "default",
					Description: "default description",
					Type:        "dynamic",
					IsNullable:  true,
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			got := marshalFunction(test.Input)

			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("mismatch of function signature: %s", diff)
			}
		})
	}
}
