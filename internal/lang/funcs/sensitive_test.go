// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestSensitive(t *testing.T) {
	tests := []struct {
		Input   cty.Value
		WantErr string
	}{
		{
			cty.NumberIntVal(1),
			``,
		},
		{
			// Unknown values stay unknown while becoming sensitive
			cty.UnknownVal(cty.String),
			``,
		},
		{
			// Null values stay unknown while becoming sensitive
			cty.NullVal(cty.String),
			``,
		},
		{
			// DynamicVal can be marked as sensitive
			cty.DynamicVal,
			``,
		},
		{
			// The marking is shallow only
			cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
			``,
		},
		{
			// A value already marked is allowed and stays marked
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			``,
		},
		{
			// A value deep already marked is allowed and stays marked,
			// _and_ we'll also mark the outer collection as sensitive.
			cty.ListVal([]cty.Value{cty.NumberIntVal(1).Mark(marks.Sensitive)}),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := Sensitive(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.HasMark(marks.Sensitive) {
				t.Errorf("result is not marked sensitive")
			}

			gotRaw, _ := got.Unmark()

			// Disregarding shallow marks, the result should have the same
			// effective value as the input.
			wantRaw, _ := test.Input.Unmark()
			if !gotRaw.RawEquals(wantRaw) {
				t.Errorf("wrong unmarked result\ngot:  %#v\nwant: %#v", got, wantRaw)
			}
		})
	}
}

func TestNonsensitive(t *testing.T) {
	tests := []struct {
		Input   cty.Value
		WantErr string
	}{
		{
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			``,
		},
		{
			cty.DynamicVal.Mark(marks.Sensitive),
			``,
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			``,
		},
		{
			cty.NullVal(cty.EmptyObject).Mark(marks.Sensitive),
			``,
		},
		{
			// The inner sensitive remains afterwards
			cty.ListVal([]cty.Value{cty.NumberIntVal(1).Mark(marks.Sensitive)}).Mark(marks.Sensitive),
			``,
		},

		// Passing a value that is already non-sensitive is not an error,
		// as this function may be used with specific to ensure that all
		// values are indeed non-sensitive
		{
			cty.NumberIntVal(1),
			``,
		},
		{
			cty.NullVal(cty.String),
			``,
		},

		// Unknown values may become sensitive once they are known, so we
		// permit them to be marked nonsensitive.
		{
			cty.DynamicVal,
			``,
		},
		{
			cty.UnknownVal(cty.String),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("nonsensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := Nonsensitive(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.HasMark(marks.Sensitive) {
				t.Errorf("result is still marked sensitive")
			}
			wantRaw, _ := test.Input.Unmark()
			if !got.RawEquals(wantRaw) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Input)
			}
		})
	}
}

func TestIssensitive(t *testing.T) {
	tests := []struct {
		Input     cty.Value
		Sensitive cty.Value
		WantErr   string
	}{
		{
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			cty.True,
			``,
		},
		{
			cty.NumberIntVal(1),
			cty.False,
			``,
		},
		{
			cty.DynamicVal.Mark(marks.Sensitive),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			cty.True,
			``,
		},
		{
			cty.NullVal(cty.EmptyObject).Mark(marks.Sensitive),
			cty.True,
			``,
		},
		{
			cty.NullVal(cty.String),
			cty.False,
			``,
		},
		{
			cty.DynamicVal,
			cty.UnknownVal(cty.Bool),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.Bool),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("issensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := Issensitive(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Sensitive) {
				t.Errorf("wrong result \ngot:  %#v\nwant: %#v", got, test.Sensitive)
			}
		})
	}

}
