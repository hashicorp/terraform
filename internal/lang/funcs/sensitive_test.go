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
			// A value with some non-standard mark gets "fixed" to be marked
			// with the standard "sensitive" mark. (This situation occurring
			// would imply an inconsistency/bug elsewhere, so we're just
			// being robust about it here.)
			cty.NumberIntVal(1).Mark("bloop"),
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

			gotRaw, gotMarks := got.Unmark()
			if len(gotMarks) != 1 {
				// We're only expecting to have the "sensitive" mark we checked
				// above. Any others are an error, even if they happen to
				// appear alongside "sensitive". (We might change this rule
				// if someday we decide to use marks for some additional
				// unrelated thing in Terraform, but currently we assume that
				// _all_ marks imply sensitive, and so returning any other
				// marks would be confusing.)
				t.Errorf("extraneous marks %#v", gotMarks)
			}

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

		// Passing a value that is already non-sensitive is an error,
		// because this function should always be used with specific
		// intention, not just as a "make everything visible" hammer.
		{
			cty.NumberIntVal(1),
			`the given value is not sensitive, so this call is redundant`,
		},
		{
			cty.NullVal(cty.String),
			`the given value is not sensitive, so this call is redundant`,
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
