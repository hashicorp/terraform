// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package json

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func TestDescribeFunction(t *testing.T) {
	// NOTE: This test case is referring to some real functions in other
	// packages. and so if those functions change signature later it will
	// probably make some cases here fail. If that is the cause of the failure,
	// it's fine to update the test here to match rather than to revert the
	// change to the function signature, as long as the change to the
	// function signature is otherwise within the bounds of our compatibility
	// promises.

	tests := map[string]struct {
		Function function.Function
		Want     *Function
	}{
		"upper": {
			Function: stdlib.UpperFunc,
			Want: &Function{
				Name: "upper",
				Params: []FunctionParam{
					{
						Name: "str",
						Type: json.RawMessage(`"string"`),
					},
				},
				ReturnType: json.RawMessage(`"string"`),
			},
		},
		"coalesce": {
			Function: stdlib.CoalesceFunc,
			Want: &Function{
				Name:   "coalesce",
				Params: []FunctionParam{},
				VariadicParam: &FunctionParam{
					Name: "vals",
					Type: json.RawMessage(`"dynamic"`),
				},
				ReturnType: json.RawMessage(`"dynamic"`),
			},
		},
		"join": {
			Function: stdlib.JoinFunc,
			Want: &Function{
				Name: "join",
				Params: []FunctionParam{
					{
						Name: "separator",
						Type: json.RawMessage(`"string"`),
					},
				},
				VariadicParam: &FunctionParam{
					Name: "lists",
					Type: json.RawMessage(`["list","string"]`),
				},
				ReturnType: json.RawMessage(`"string"`),
			},
		},
		"jsonencode": {
			Function: stdlib.JSONEncodeFunc,
			Want: &Function{
				Name: "jsonencode",
				Params: []FunctionParam{
					{
						Name: "val",
						Type: json.RawMessage(`"dynamic"`),
					},
				},
				ReturnType: json.RawMessage(`"string"`),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := DescribeFunction(name, test.Function)
			want := test.Want

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
