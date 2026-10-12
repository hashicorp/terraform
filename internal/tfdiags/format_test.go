// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"fmt"
	"strings"
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

func TestFormatValueStr(t *testing.T) {
	deprecatedObj := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("example"),
	}).Mark(marks.Deprecation)

	sensitiveObj := cty.ObjectVal(map[string]cty.Value{
		"username": cty.StringVal("admin"),
		"password": cty.StringVal("hunter2").Mark(marks.Sensitive),
	})

	tests := []struct {
		Name     string
		Val      cty.Value
		WantSub  []string
		NotWant  []string
		WantFail bool
	}{
		{
			Name: "plain value",
			Val: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("example"),
			}),
			WantSub: []string{`"id": "example"`},
		},
		{
			// https://github.com/hashicorp/terraform/issues/39365: values
			// carrying non-sensitive marks (such as provider deprecation
			// marks) must render instead of failing JSON marshalling.
			Name:    "deprecated mark renders",
			Val:     deprecatedObj,
			WantSub: []string{`"id": "example"`},
		},
		{
			Name:    "sensitive stays redacted",
			Val:     sensitiveObj,
			WantSub: []string{`"username": "admin"`, "(sensitive value)"},
			NotWant: []string{"hunter2"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got, err := FormatValueStr(test.Val)
			if test.WantFail {
				if err == nil {
					t.Fatalf("expected failure, got:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			for _, want := range test.WantSub {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in:\n%s", want, got)
				}
			}
			for _, notWant := range test.NotWant {
				if strings.Contains(got, notWant) {
					t.Errorf("leaked %q in:\n%s", notWant, got)
				}
			}
		})
	}
}
