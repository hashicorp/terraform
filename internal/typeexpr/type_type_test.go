package typeexpr

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTypeConstraintType(t *testing.T) {
	tyVal1 := TypeConstraintVal(cty.String)
	tyVal2 := TypeConstraintVal(cty.String)
	tyVal3 := TypeConstraintVal(cty.Number)

	if !tyVal1.RawEquals(tyVal2) {
		t.Errorf("tyVal1 not equal to tyVal2\ntyVal1: %#v\ntyVal2: %#v", tyVal1, tyVal2)
	}
	if tyVal1.RawEquals(tyVal3) {
		t.Errorf("tyVal1 equal to tyVal2, but should not be\ntyVal1: %#v\ntyVal3: %#v", tyVal1, tyVal3)
	}

	if got, want := TypeConstraintFromVal(tyVal1), cty.String; !got.Equals(want) {
		t.Errorf("wrong type extracted from tyVal1\ngot:  %#v\nwant: %#v", got, want)
	}
	if got, want := TypeConstraintFromVal(tyVal3), cty.Number; !got.Equals(want) {
		t.Errorf("wrong type extracted from tyVal3\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestConvertFunc(t *testing.T) {
	// This is testing the convert function directly, skipping over the HCL
	// parsing and evaluation steps that would normally lead there. There is
	// another test in the "integrationtest" package called TestTypeConvertFunc
	// that exercises the full path to this function via the hclsyntax parser.

	tests := []struct {
		val, ty cty.Value
		want    cty.Value
		wantErr string
	}{
		// The goal here is not an exhaustive set of conversions, since that's
		// already covered in cty/convert, but rather exercising different
		// permutations of success and failure to make sure the function
		// handles all of the results in a reasonable way.
		{
			cty.StringVal("hello"),
			TypeConstraintVal(cty.String),
			cty.StringVal("hello"),
			``,
		},
		{
			cty.True,
			TypeConstraintVal(cty.String),
			cty.StringVal("true"),
			``,
		},
		{
			cty.StringVal("hello"),
			TypeConstraintVal(cty.Bool),
			cty.NilVal,
			`a bool is required`,
		},
		{
			cty.UnknownVal(cty.Bool),
			TypeConstraintVal(cty.Bool),
			cty.UnknownVal(cty.Bool),
			``,
		},
		{
			cty.DynamicVal,
			TypeConstraintVal(cty.Bool),
			cty.UnknownVal(cty.Bool),
			``,
		},
		{
			cty.NullVal(cty.Bool),
			TypeConstraintVal(cty.Bool),
			cty.NullVal(cty.Bool),
			``,
		},
		{
			cty.NullVal(cty.DynamicPseudoType),
			TypeConstraintVal(cty.Bool),
			cty.NullVal(cty.Bool),
			``,
		},
		{
			cty.StringVal("hello").Mark(1),
			TypeConstraintVal(cty.String),
			cty.StringVal("hello").Mark(1),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v to %#v", test.val, test.ty), func(t *testing.T) {
			got, err := ConvertFunc.Call([]cty.Value{test.val, test.ty})

			if err != nil {
				if test.wantErr != "" {
					if got, want := err.Error(), test.wantErr; got != want {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
					}
				} else {
					t.Errorf("unexpected error\ngot:  %s\nwant: <nil>", err)
				}
				return
			}
			if test.wantErr != "" {
				t.Errorf("wrong error\ngot:  <nil>\nwant: %s", test.wantErr)
			}

			if !test.want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
