// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonfunction

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		Name              string
		Functions         map[string]function.Function
		ProviderFunctions map[string]providers.FunctionDecl
		Want              string
		WantErr           string
	}{
		{
			Name: "minimal function",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
					Type: function.StaticReturnType(cty.Bool),
				}),
			},
			ProviderFunctions: map[string]providers.FunctionDecl{
				"fun": {
					ReturnType: cty.Bool,
				},
			},
			Want: `{"format_version":"1.0","function_signatures":{"fun":{"return_type":"bool"}}}`,
		},
		{
			Name: "function with description",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
					Description: "`timestamp` returns a UTC timestamp string.",
					Type:        function.StaticReturnType(cty.String),
				}),
			},
			ProviderFunctions: map[string]providers.FunctionDecl{
				"fun": {
					Description: "`timestamp` returns a UTC timestamp string.",
					ReturnType:  cty.String,
				},
			},
			Want: "{\"format_version\":\"1.0\",\"function_signatures\":{\"fun\":{\"description\":\"`timestamp` returns a UTC timestamp string.\",\"return_type\":\"string\"}}}",
		},
		{
			Name: "function with parameters",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
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
			},
			ProviderFunctions: map[string]providers.FunctionDecl{
				"fun": {
					Parameters: []providers.FunctionParam{
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
					ReturnType: cty.String,
				},
			},
			Want: `{"format_version":"1.0","function_signatures":{"fun":{"return_type":"string","parameters":[{"name":"timestamp","description":"timestamp text","type":"string"},{"name":"duration","description":"duration text","type":"string"}]}}}`,
		},
		{
			Name: "function with variadic parameter",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
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
			},
			ProviderFunctions: map[string]providers.FunctionDecl{
				"fun": {
					VariadicParameter: &providers.FunctionParam{
						Name:               "default",
						Description:        "default description",
						Type:               cty.DynamicPseudoType,
						AllowUnknownValues: true,
						AllowNullValue:     true,
					},
					ReturnType: cty.DynamicPseudoType,
				},
			},
			Want: `{"format_version":"1.0","function_signatures":{"fun":{"return_type":"dynamic","variadic_parameter":{"name":"default","description":"default description","is_nullable":true,"type":"dynamic"}}}}`,
		},
		{
			Name: "function with list types",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
					Params: []function.Parameter{
						{
							Name: "list",
							Type: cty.List(cty.String),
						},
					},
					Type: function.StaticReturnType(cty.List(cty.String)),
				}),
			},
			ProviderFunctions: map[string]providers.FunctionDecl{
				"fun": {
					Parameters: []providers.FunctionParam{
						{
							Name: "list",
							Type: cty.List(cty.String),
						},
					},
					ReturnType: cty.List(cty.String),
				},
			},
			Want: `{"format_version":"1.0","function_signatures":{"fun":{"return_type":["list","string"],"parameters":[{"name":"list","type":["list","string"]}]}}}`,
		},
		{
			Name: "returns diagnostics on failure",
			Functions: map[string]function.Function{
				"fun": function.New(&function.Spec{
					Params: []function.Parameter{},
					Type: func(args []cty.Value) (ret cty.Type, err error) {
						return cty.DynamicPseudoType, fmt.Errorf("error")
					},
				}),
			},
			WantErr: "Failed to serialize function \"fun\": error",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			got, diags := Marshal(test.Functions)
			if test.WantErr != "" {
				if !diags.HasErrors() {
					t.Fatal("expected error, got none")
				}
				if diags.Err().Error() != test.WantErr {
					t.Fatalf("expected error %q, got %q", test.WantErr, diags.Err())
				}
			} else {
				if diags.HasErrors() {
					t.Fatal(diags)
				}

				if diff := cmp.Diff(test.Want, string(got)); diff != "" {
					t.Fatalf("mismatch of function signature: %s", diff)
				}
			}

			if test.ProviderFunctions != nil {
				// Provider functions should marshal identically to cty
				// functions, without the wrapping object.
				got := MarshalProviderFunctions(test.ProviderFunctions)

				gotBytes, err := json.Marshal(got)

				if err != nil {
					// these should never error
					t.Fatal("Marshal of ProviderFunctions failed:", err)
				}

				var want functions

				err = json.Unmarshal([]byte(test.Want), &want)

				if err != nil {
					// these should never error
					t.Fatal("Unmarshal of Want failed:", err)
				}

				wantBytes, err := json.Marshal(want.Signatures)

				if err != nil {
					// these should never error
					t.Fatal("Marshal of Want.Signatures failed:", err)
				}

				if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
					t.Fatalf("mismatch of function signature: %s", diff)
				}
			}
		})
	}
}
