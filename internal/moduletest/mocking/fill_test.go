// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"math/rand"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestFillType(t *testing.T) {
	tcs := map[string]struct {
		in  cty.Value
		out cty.Value
	}{
		"object_to_object": {
			in: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("hello"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},
		"map_to_object": {
			in: cty.MapVal(map[string]cty.Value{
				"id": cty.StringVal("hello"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("hello"),
				"value": cty.StringVal("ssnk9qhr"),
			}),
		},
		"list_to_list": {
			in: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"tuple_to_list": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"set_to_list": {
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
			out: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("amyllmyg"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("amyllmyg"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},
		"list_to_set": {
			in: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"tuple_to_set": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"set_to_set": {
			in: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
			out: cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("amyllmyg"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("amyllmyg"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},
		"tuple_to_tuple": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"map_to_map": {
			in: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{}),
				"two": cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				"two": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"object_to_map": {
			in: cty.ObjectVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{}),
				"two": cty.ObjectVal(map[string]cty.Value{}),
			}),
			out: cty.MapVal(map[string]cty.Value{
				"one": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("ssnk9qhr"),
				}),
				"two": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"additional_attributes": {
			in: cty.ObjectVal(map[string]cty.Value{
				"one": cty.StringVal("hello"),
				"two": cty.StringVal("world"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"one":   cty.StringVal("hello"),
				"three": cty.StringVal("ssnk9qhr"),
			}),
		},
		// This is just a sort of safety check to validate it falls through to
		// normal conversions for everything we don't handle.
		"normal_conversion": {
			in: cty.MapVal(map[string]cty.Value{
				"key_one": cty.StringVal("value_one"),
				"key_two": cty.StringVal("value_two"),
			}),
			out: cty.ObjectVal(map[string]cty.Value{
				"key_one": cty.StringVal("value_one"),
				"key_two": cty.StringVal("value_two"),
			}),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			// Let's have predictable test outcomes.
			testRand = rand.New(rand.NewSource(0))
			defer func() {
				testRand = nil
			}()

			actual, err := FillType(tc.in, tc.out.Type())
			if err != nil {
				t.Fatal(err)
			}

			expected := tc.out
			if !expected.RawEquals(actual) {
				t.Errorf("expected:%s\nactual:   %s", expected.GoString(), actual.GoString())
			}
		})
	}
}

func TestFillType_Errors(t *testing.T) {

	tcs := map[string]struct {
		in     cty.Value
		target cty.Type
		err    string
	}{
		"error_diff_tuple_types": {
			in: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{}),
				cty.StringVal("not an object"),
			}),
			target: cty.List(cty.EmptyObject),
			err:    "incompatible types; expected object, found string",
		},
		"error_diff_object_types": {
			in: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{}),
				"string": cty.StringVal("not an object"),
			}),
			target: cty.Map(cty.EmptyObject),
			err:    "incompatible types; expected object, found string",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			actual, err := FillType(tc.in, tc.target)
			if err == nil {
				t.Fatal("should have errored")
			}

			if out := err.Error(); out != tc.err {
				t.Errorf("\nexpected: %s\nactual:   %s", tc.err, out)
			}

			if actual != cty.NilVal {
				t.Fatal("should have errored")
			}
		})
	}

}
