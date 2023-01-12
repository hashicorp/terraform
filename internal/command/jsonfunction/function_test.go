package jsonfunction

import (
	"encoding/json"
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
				ReturnType: json.RawMessage(`"bool"`),
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
				ReturnType:  json.RawMessage(`"string"`),
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
				ReturnType: json.RawMessage(`"string"`),
				Parameters: []*parameter{
					{
						Name:        "timestamp",
						Description: "timestamp text",
						Type:        json.RawMessage(`"string"`),
					},
					{
						Name:        "duration",
						Description: "duration text",
						Type:        json.RawMessage(`"string"`),
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
				ReturnType: json.RawMessage(`"dynamic"`),
				VariadicParameter: &parameter{
					Name:        "default",
					Description: "default description",
					Type:        json.RawMessage(`"dynamic"`),
					IsNullable:  true,
				},
			},
		},
		{
			"function with list types",
			function.New(&function.Spec{
				Params: []function.Parameter{
					{
						Name: "list",
						Type: cty.List(cty.String),
					},
				},
				Type: function.StaticReturnType(cty.List(cty.String)),
			}),
			&FunctionSignature{
				ReturnType: json.RawMessage(`["list","string"]`),
				Parameters: []*parameter{
					{
						Name: "list",
						Type: json.RawMessage(`["list","string"]`),
					},
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			got, err := marshalFunction(test.Input)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("mismatch of function signature: %s", diff)
			}
		})
	}
}
