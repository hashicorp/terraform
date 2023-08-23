// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestReplace(t *testing.T) {
	tests := []struct {
		String  cty.Value
		Substr  cty.Value
		Replace cty.Value
		Want    cty.Value
		Err     bool
	}{
		{ // Regular search and replace
			cty.StringVal("hello"),
			cty.StringVal("hel"),
			cty.StringVal("bel"),
			cty.StringVal("bello"),
			false,
		},
		{ // Search string doesn't match
			cty.StringVal("hello"),
			cty.StringVal("nope"),
			cty.StringVal("bel"),
			cty.StringVal("hello"),
			false,
		},
		{ // Regular expression
			cty.StringVal("hello"),
			cty.StringVal("/l/"),
			cty.StringVal("L"),
			cty.StringVal("heLLo"),
			false,
		},
		{
			cty.StringVal("helo"),
			cty.StringVal("/(l)/"),
			cty.StringVal("$1$1"),
			cty.StringVal("hello"),
			false,
		},
		{ // Bad regexp
			cty.StringVal("hello"),
			cty.StringVal("/(l/"),
			cty.StringVal("$1$1"),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("replace(%#v, %#v, %#v)", test.String, test.Substr, test.Replace), func(t *testing.T) {
			got, err := Replace(test.String, test.Substr, test.Replace)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestStrContains(t *testing.T) {
	tests := []struct {
		String cty.Value
		Substr cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("hel"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello"),
			cty.StringVal("lo"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.StringVal("1"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.StringVal("heo"),
			cty.BoolVal(false),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.NumberIntVal(1),
			cty.UnknownVal(cty.Bool),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("includes(%#v, %#v)", test.String, test.Substr), func(t *testing.T) {
			got, err := StrContains(test.String, test.Substr)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestStartsWith(t *testing.T) {
	tests := []struct {
		String, Prefix cty.Value
		Want           cty.Value
		WantError      string
	}{
		{
			cty.StringVal("hello world"),
			cty.StringVal("hello"),
			cty.True,
			``,
		},
		{
			cty.StringVal("hey world"),
			cty.StringVal("hello"),
			cty.False,
			``,
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.StringVal("a"),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.StringVal(""),
			cty.StringVal("a"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("a"),
			cty.UnknownVal(cty.Bool).RefineNotNull(),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("a"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("ht"),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https:"),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https-"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https://"),
			cty.UnknownVal(cty.Bool).RefineNotNull(),
			``,
		},
		{
			// Unicode combining characters edge-case: we match the prefix
			// in terms of unicode code units rather than grapheme clusters,
			// which is inconsistent with our string processing elsewhere but
			// would be a breaking change to fix that bug now.
			cty.StringVal("\U0001f937\u200d\u2642"), // "Man Shrugging" is encoded as "Person Shrugging" followed by zero-width joiner and then the masculine gender presentation modifier
			cty.StringVal("\U0001f937"),             // Just the "Person Shrugging" character without any modifiers
			cty.True,
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("StartsWith(%#v, %#v)", test.String, test.Prefix), func(t *testing.T) {
			got, err := StartsWithFunc.Call([]cty.Value{test.String, test.Prefix})

			if test.WantError != "" {
				gotErr := fmt.Sprintf("%s", err)
				if gotErr != test.WantError {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantError)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf(
					"wrong result\nstring: %#v\nprefix: %#v\ngot:    %#v\nwant:   %#v",
					test.String, test.Prefix, got, test.Want,
				)
			}
		})
	}
}
