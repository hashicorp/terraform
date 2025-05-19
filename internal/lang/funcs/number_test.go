// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestLog(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Base cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
			false,
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(1),
			false,
		},

		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(10),
			cty.NegativeInfinity,
			false,
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(-0),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("log(%#v, %#v)", test.Num, test.Base), func(t *testing.T) {
			got, err := Log(test.Num, test.Base)

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

func TestPow(t *testing.T) {
	tests := []struct {
		Num   cty.Value
		Power cty.Value
		Want  cty.Value
		Err   bool
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
			false,
		},

		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(2),
			false,
		},
		{
			cty.NumberFloatVal(3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
			false,
		},
		{
			cty.NumberFloatVal(-3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
			false,
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(-2),
			cty.NumberFloatVal(0.25),
			false,
		},
		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("pow(%#v, %#v)", test.Num, test.Power), func(t *testing.T) {
			got, err := Pow(test.Num, test.Power)

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

func TestSignum(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(0),
			false,
		},
		{
			cty.NumberFloatVal(12),
			cty.NumberFloatVal(1),
			false,
		},
		{
			cty.NumberFloatVal(-29),
			cty.NumberFloatVal(-1),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("signum(%#v)", test.Num), func(t *testing.T) {
			got, err := Signum(test.Num)

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

func TestParseInt(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Base cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.StringVal("128"),
			cty.NumberIntVal(10),
			cty.NumberIntVal(128),
			``,
		},
		{
			cty.StringVal("128").Mark(marks.Sensitive),
			cty.NumberIntVal(10),
			cty.NumberIntVal(128).Mark(marks.Sensitive),
			``,
		},
		{
			cty.StringVal("128"),
			cty.NumberIntVal(10).Mark(marks.Sensitive),
			cty.NumberIntVal(128).Mark(marks.Sensitive),
			``,
		},
		{
			cty.StringVal("128").Mark(marks.Sensitive),
			cty.NumberIntVal(10).Mark(marks.Sensitive),
			cty.NumberIntVal(128).Mark(marks.Sensitive),
			``,
		},
		{
			cty.StringVal("128").Mark(marks.Sensitive),
			cty.UnknownVal(cty.Number).Mark(marks.Sensitive),
			cty.UnknownVal(cty.Number).RefineNotNull().Mark(marks.Sensitive),
			``,
		},
		{
			cty.StringVal("128").Mark("boop"),
			cty.NumberIntVal(10).Mark(marks.Sensitive),
			cty.NumberIntVal(128).WithMarks(cty.NewValueMarks("boop", marks.Sensitive)),
			``,
		},
		{
			cty.StringVal("-128"),
			cty.NumberIntVal(10),
			cty.NumberIntVal(-128),
			``,
		},
		{
			cty.StringVal("00128"),
			cty.NumberIntVal(10),
			cty.NumberIntVal(128),
			``,
		},
		{
			cty.StringVal("-00128"),
			cty.NumberIntVal(10),
			cty.NumberIntVal(-128),
			``,
		},
		{
			cty.StringVal("FF00"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(65280),
			``,
		},
		{
			cty.StringVal("ff00"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(65280),
			``,
		},
		{
			cty.StringVal("-FF00"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(-65280),
			``,
		},
		{
			cty.StringVal("00FF00"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(65280),
			``,
		},
		{
			cty.StringVal("-00FF00"),
			cty.NumberIntVal(16),
			cty.NumberIntVal(-65280),
			``,
		},
		{
			cty.StringVal("1011111011101111"),
			cty.NumberIntVal(2),
			cty.NumberIntVal(48879),
			``,
		},
		{
			cty.StringVal("aA"),
			cty.NumberIntVal(62),
			cty.NumberIntVal(656),
			``,
		},
		{
			cty.StringVal("Aa"),
			cty.NumberIntVal(62),
			cty.NumberIntVal(2242),
			``,
		},
		{
			cty.StringVal("999999999999999999999999999999999999999999999999999999999999"),
			cty.NumberIntVal(10),
			cty.MustParseNumberVal("999999999999999999999999999999999999999999999999999999999999"),
			``,
		},
		{
			cty.StringVal("FF"),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`cannot parse "FF" as a base 10 integer`,
		},
		{
			cty.StringVal("FF").Mark(marks.Sensitive),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`cannot parse (sensitive value) as a base 10 integer`,
		},
		{
			cty.StringVal("FF").Mark(marks.Sensitive),
			cty.NumberIntVal(10).Mark(marks.Sensitive),
			cty.UnknownVal(cty.Number),
			`cannot parse (sensitive value) as a base (sensitive value) integer`,
		},
		{
			cty.StringVal("00FF"),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`cannot parse "00FF" as a base 10 integer`,
		},
		{
			cty.StringVal("-00FF"),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`cannot parse "-00FF" as a base 10 integer`,
		},
		{
			cty.NumberIntVal(2),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`first argument must be a string, not number`,
		},
		{
			cty.StringVal("1"),
			cty.NumberIntVal(63),
			cty.UnknownVal(cty.Number),
			`base must be a whole number between 2 and 62 inclusive`,
		},
		{
			cty.StringVal("1"),
			cty.NumberIntVal(-1),
			cty.UnknownVal(cty.Number),
			`base must be a whole number between 2 and 62 inclusive`,
		},
		{
			cty.StringVal("1"),
			cty.NumberIntVal(1),
			cty.UnknownVal(cty.Number),
			`base must be a whole number between 2 and 62 inclusive`,
		},
		{
			cty.StringVal("1"),
			cty.NumberIntVal(0),
			cty.UnknownVal(cty.Number),
			`base must be a whole number between 2 and 62 inclusive`,
		},
		{
			cty.StringVal("1.2"),
			cty.NumberIntVal(10),
			cty.UnknownVal(cty.Number),
			`cannot parse "1.2" as a base 10 integer`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("parseint(%#v, %#v)", test.Num, test.Base), func(t *testing.T) {
			got, err := ParseInt(test.Num, test.Base)

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.Err; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
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
