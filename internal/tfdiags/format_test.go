// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestCompactValueStr(t *testing.T) {
	tests := []struct {
		Val  cty.Value
		Want string
	}{
		{
			cty.NullVal(cty.DynamicPseudoType),
			"null",
		},
		{
			cty.UnknownVal(cty.DynamicPseudoType),
			"(not yet known)",
		},
		{
			cty.False,
			"false",
		},
		{
			cty.True,
			"true",
		},
		{
			cty.NumberIntVal(5),
			"5",
		},
		{
			cty.NumberFloatVal(5.2),
			"5.2",
		},
		{
			cty.StringVal(""),
			`""`,
		},
		{
			cty.StringVal("hello"),
			`"hello"`,
		},
		{
			cty.ListValEmpty(cty.String),
			"empty list of string",
		},
		{
			cty.SetValEmpty(cty.String),
			"empty set of string",
		},
		{
			cty.EmptyTupleVal,
			"empty tuple",
		},
		{
			cty.MapValEmpty(cty.String),
			"empty map of string",
		},
		{
			cty.EmptyObjectVal,
			"object with no attributes",
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			"list of string with 1 element",
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			"list of string with 2 elements",
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			`object with 1 attribute "a"`,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
				"c": cty.StringVal("d"),
			}),
			"object with 2 attributes",
		},
		{
			cty.StringVal("a sensitive value").Mark(marks.Sensitive),
			"(sensitive value)",
		},
		{
			cty.StringVal("an ephemeral value").Mark(marks.Ephemeral),
			"(ephemeral value)",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Val), func(t *testing.T) {
			got := CompactValueStr(test.Val)
			if got != test.Want {
				t.Errorf("wrong result\nvalue: %#v\ngot:   %s\nwant:  %s", test.Val, got, test.Want)
			}
		})
	}
}
