// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestLog_valid(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Base cty.Value
		Want cty.Value
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(1),
		},

		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(10),
			cty.NegativeInfinity,
		},
		{
			cty.NumberFloatVal(10),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(-0),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("log(%#v, %#v)", test.Num, test.Base), func(t *testing.T) {
			got, err := LogFunc.Call([]cty.Value{test.Num, test.Base})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestLog_invalid(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Base cty.Value
	}{
		{
			// NaN result
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
		},
		{
			// NaN result
			cty.NumberFloatVal(-1),
			cty.NumberFloatVal(10),
		},
		// {
		// 	// -Infinity result: log(0, 10)
		// 	cty.NumberFloatVal(0),
		// 	cty.NumberFloatVal(10),
		// },
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("log(%#v, %#v)", test.Num, test.Base), func(t *testing.T) {
			_, err := LogFunc.Call([]cty.Value{test.Num, test.Base})

			if err == nil {
				t.Fatal("expected error but got none")
			}

			expectedErr := "result is not a number"
			if err.Error() != expectedErr {
				t.Fatal(err)
			}
		})
	}
}

func TestPow_valid(t *testing.T) {
	tests := []struct {
		Num   cty.Value
		Power cty.Value
		Want  cty.Value
	}{
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
		},
		{
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(1),
		},

		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(1),
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(1),
			cty.NumberFloatVal(2),
		},
		{
			cty.NumberFloatVal(3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
		},
		{
			cty.NumberFloatVal(-3),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(9),
		},
		{
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(-2),
			cty.NumberFloatVal(0.25),
		},
		{
			cty.NumberFloatVal(0),
			cty.NumberFloatVal(2),
			cty.NumberFloatVal(0),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("pow(%#v, %#v)", test.Num, test.Power), func(t *testing.T) {
			got, err := PowFunc.Call([]cty.Value{test.Num, test.Power})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestPow_invalid(t *testing.T) {
	tests := []struct {
		Num   cty.Value
		Power cty.Value
	}{
		{
			cty.NumberFloatVal(-2),
			cty.NumberFloatVal(0.5),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("pow(%#v, %#v)", test.Num, test.Power), func(t *testing.T) {
			_, err := PowFunc.Call([]cty.Value{test.Num, test.Power})
			if err == nil {
				t.Fatal("succeeded; want error")
			}

			expectedErr := "result is not a number"
			if err.Error() != expectedErr {
				t.Fatal(err)
			}
		})
	}
}
