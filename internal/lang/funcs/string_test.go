// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func StrContains(str, substr cty.Value) (cty.Value, error) {
	return StrContainsFunc.Call([]cty.Value{str, substr})
}
