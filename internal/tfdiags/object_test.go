// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func Test_ObjectToString(t *testing.T) {
	testCases := []struct {
		name     string
		value    cty.Value
		expected string
	}{
		{
			name:     "null",
			value:    cty.NullVal(cty.Object(map[string]cty.Type{})),
			expected: "<null>",
		},
		{
			name:     "unknown",
			value:    cty.UnknownVal(cty.Object(map[string]cty.Type{})),
			expected: "<unknown>",
		},
		{
			name:     "empty",
			value:    cty.EmptyObjectVal,
			expected: "<empty>",
		},
		{
			name: "primitive",
			value: cty.ObjectVal(map[string]cty.Value{
				"number": cty.NumberIntVal(42),
				"string": cty.StringVal("hello"),
				"bool":   cty.BoolVal(true),
			}),
			expected: "bool=true,number=42,string=hello",
		},
		{
			name: "list",
			value: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("hello"),
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
			}),
			expected: "list=[a,b,c],string=hello",
		},
		{
			name: "with null value",
			value: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("hello"),
				"null":   cty.NullVal(cty.String),
			}),
			expected: "null=<null>,string=hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ObjectToString(tc.value)

			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
