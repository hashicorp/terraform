package list

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTranspose(t *testing.T) {
	tests := []struct {
		input     cty.Value
		expected  cty.Value
		wantError bool
	}{
		{
			input: cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a1"),
					cty.StringVal("a2"),
					cty.StringVal("a3"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("b1"),
					cty.StringVal("b2"),
					cty.StringVal("b3"),
				}),
			}),
			expected: cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("a1"),
					cty.StringVal("b1"),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("a2"),
					cty.StringVal("b2"),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("a3"),
					cty.StringVal("b3"),
				}),
			}),
			wantError: false,
		},
		{
			// Test with uneven lists
			input: cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("a1"),
					cty.StringVal("a2"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("b1"),
				}),
			}),
			expected: cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("a1"),
					cty.StringVal("b1"),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("a2"),
					cty.NullVal(cty.String),
				}),
			}),
			wantError: false,
		},
		{
			// Test empty list
			input:     cty.ListValEmpty(cty.List(cty.String)),
			expected:  cty.ListValEmpty(cty.DynamicPseudoType),
			wantError: false,
		},
	}

	for _, test := range tests {
		t.Run("transpose", func(t *testing.T) {
			got, err := TransposeFunc().Call([]cty.Value{test.input})

			if test.wantError {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.expected) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.expected)
			}
		})
	}
}
