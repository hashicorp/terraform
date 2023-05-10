// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcl2shim

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestValuesSDKEquivalent(t *testing.T) {
	piBig, _, err := big.ParseFloat("3.14159265358979323846264338327950288419716939937510582097494459", 10, 512, big.ToZero)
	if err != nil {
		t.Fatal(err)
	}
	pi64, _ := piBig.Float64()

	tests := []struct {
		A, B cty.Value
		Want bool
	}{
		// Strings
		{
			cty.StringVal("hello"),
			cty.StringVal("hello"),
			true,
		},
		{
			cty.StringVal("hello"),
			cty.StringVal("world"),
			false,
		},
		{
			cty.StringVal("hello"),
			cty.StringVal(""),
			false,
		},
		{
			cty.NullVal(cty.String),
			cty.StringVal(""),
			true,
		},

		// Numbers
		{
			cty.NumberIntVal(1),
			cty.NumberIntVal(1),
			true,
		},
		{
			cty.NumberIntVal(1),
			cty.NumberIntVal(2),
			false,
		},
		{
			cty.NumberIntVal(1),
			cty.Zero,
			false,
		},
		{
			cty.NullVal(cty.Number),
			cty.Zero,
			true,
		},
		{
			cty.NumberVal(piBig),
			cty.Zero,
			false,
		},
		{
			cty.NumberFloatVal(pi64),
			cty.Zero,
			false,
		},
		{
			cty.NumberFloatVal(pi64),
			cty.NumberVal(piBig),
			true,
		},

		// Bools
		{
			cty.True,
			cty.True,
			true,
		},
		{
			cty.True,
			cty.False,
			false,
		},
		{
			cty.NullVal(cty.Bool),
			cty.False,
			true,
		},

		// Mixed primitives
		{
			cty.StringVal("hello"),
			cty.False,
			false,
		},
		{
			cty.StringVal(""),
			cty.False,
			true,
		},
		{
			cty.NumberIntVal(0),
			cty.False,
			true,
		},
		{
			cty.StringVal(""),
			cty.NumberIntVal(0),
			true,
		},
		{
			cty.NullVal(cty.Bool),
			cty.NullVal(cty.Number),
			true,
		},
		{
			cty.StringVal(""),
			cty.NullVal(cty.Number),
			true,
		},

		// Lists
		{
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
			true,
		},
		{
			cty.ListValEmpty(cty.String),
			cty.NullVal(cty.List(cty.String)),
			true,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("hello")}),
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			true,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("hello")}),
			cty.ListVal([]cty.Value{cty.StringVal("world")}),
			false,
		},
		{
			cty.ListVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.ListVal([]cty.Value{cty.StringVal("")}),
			true,
		},

		// Tuples
		{
			cty.EmptyTupleVal,
			cty.EmptyTupleVal,
			true,
		},
		{
			cty.EmptyTupleVal,
			cty.NullVal(cty.EmptyTuple),
			true,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("hello")}),
			false,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			cty.EmptyTupleVal,
			false,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			true,
		},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("hello")}),
			cty.TupleVal([]cty.Value{cty.StringVal("world")}),
			false,
		},
		{
			cty.TupleVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.TupleVal([]cty.Value{cty.StringVal("")}),
			true,
		},

		// Sets
		{
			cty.SetValEmpty(cty.String),
			cty.SetValEmpty(cty.String),
			true,
		},
		{
			cty.SetValEmpty(cty.String),
			cty.NullVal(cty.Set(cty.String)),
			true,
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			true,
		},
		{
			cty.SetVal([]cty.Value{cty.StringVal("hello")}),
			cty.SetVal([]cty.Value{cty.StringVal("world")}),
			false,
		},
		{
			cty.SetVal([]cty.Value{cty.NullVal(cty.String)}),
			cty.SetVal([]cty.Value{cty.StringVal("")}),
			true,
		},
		{
			cty.SetVal([]cty.Value{
				cty.NullVal(cty.String),
				cty.StringVal(""),
			}),
			cty.SetVal([]cty.Value{
				cty.NullVal(cty.String),
			}),
			false, // because the element count is different
		},
		{
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal(""),
					"b": cty.StringVal(""),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.NullVal(cty.String),
					"b": cty.StringVal(""),
				}),
			}),
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal(""),
					"b": cty.StringVal(""),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal(""),
					"b": cty.NullVal(cty.String),
				}),
			}),
			true,
		},
		{
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("boop"),
					"b": cty.StringVal(""),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.NullVal(cty.String),
					"b": cty.StringVal(""),
				}),
			}),
			cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("beep"),
					"b": cty.StringVal(""),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal(""),
					"b": cty.NullVal(cty.String),
				}),
			}),
			false,
		},
		{
			cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(cty.String),
				"list_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"unused": cty.StringVal(""),
					}),
				}),
			})}),
			cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(cty.String),
				"list_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"unused": cty.NullVal(cty.String),
					}),
				}),
			})}),
			true,
		},

		// Maps
		{
			cty.MapValEmpty(cty.String),
			cty.MapValEmpty(cty.String),
			true,
		},
		{
			cty.MapValEmpty(cty.String),
			cty.NullVal(cty.Map(cty.String)),
			true,
		},
		{
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello"), "hey": cty.StringVal("hello")}),
			false,
		},
		{
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.MapValEmpty(cty.String),
			false,
		},
		{
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			true,
		},
		{
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("world")}),
			false,
		},
		{
			cty.MapVal(map[string]cty.Value{"hi": cty.NullVal(cty.String)}),
			cty.MapVal(map[string]cty.Value{"hi": cty.StringVal("")}),
			true,
		},

		// Objects
		{
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			true,
		},
		{
			cty.EmptyObjectVal,
			cty.NullVal(cty.EmptyObject),
			true,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello"), "hey": cty.StringVal("hello")}),
			false,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.EmptyObjectVal,
			false,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			true,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("hello")}),
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("world")}),
			false,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"hi": cty.NullVal(cty.String)}),
			cty.ObjectVal(map[string]cty.Value{"hi": cty.StringVal("")}),
			true,
		},

		// Unknown values
		{
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			true,
		},
		{
			cty.StringVal("hello"),
			cty.UnknownVal(cty.String),
			false,
		},
		{
			cty.StringVal(""),
			cty.UnknownVal(cty.String),
			false,
		},
		{
			cty.NullVal(cty.String),
			cty.UnknownVal(cty.String),
			false,
		},
	}

	run := func(t *testing.T, a, b cty.Value, want bool) {
		got := ValuesSDKEquivalent(a, b)

		if got != want {
			t.Errorf("wrong result\nfor: %#v ≈ %#v\ngot %#v, but want %#v", a, b, got, want)
		}
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v ≈ %#v", test.A, test.B), func(t *testing.T) {
			run(t, test.A, test.B, test.Want)
		})
		// This function is symmetrical, so we'll also test in reverse so
		// we don't need to manually copy all of the test cases. (But this does
		// mean that one failure normally becomes two, of course!)
		if !test.A.RawEquals(test.B) {
			t.Run(fmt.Sprintf("%#v ≈ %#v", test.B, test.A), func(t *testing.T) {
				run(t, test.B, test.A, test.Want)
			})
		}
	}
}
