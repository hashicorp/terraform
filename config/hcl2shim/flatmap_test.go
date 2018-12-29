package hcl2shim

import (
	"fmt"
	"testing"

	"github.com/go-test/deep"

	"github.com/zclconf/go-cty/cty"
)

func TestFlatmapValueFromHCL2(t *testing.T) {
	tests := []struct {
		Value cty.Value
		Want  map[string]string
	}{
		{
			cty.EmptyObjectVal,
			map[string]string{},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
			}),
			map[string]string{
				"foo": "hello",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Bool),
			}),
			map[string]string{
				"foo": UnknownVariableValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NumberIntVal(12),
			}),
			map[string]string{
				"foo": "12",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
				"bar": cty.False,
			}),
			map[string]string{
				"foo": "true",
				"bar": "false",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("world"),
				"baz": cty.StringVal("whelp"),
			}),
			map[string]string{
				"foo": "hello",
				"bar": "world",
				"baz": "whelp",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(cty.String),
			}),
			map[string]string{
				"foo.#": "0",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.List(cty.String)),
			}),
			map[string]string{
				"foo.#": UnknownVariableValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
				}),
			}),
			map[string]string{
				"foo.#": "1",
				"foo.0": "hello",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("world"),
				}),
			}),
			map[string]string{
				"foo.#": "2",
				"foo.0": "hello",
				"foo.1": "world",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"hello":       cty.NumberIntVal(12),
					"hello.world": cty.NumberIntVal(10),
				}),
			}),
			map[string]string{
				"foo.%":           "2",
				"foo.hello":       "12",
				"foo.hello.world": "10",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Map(cty.String)),
			}),
			map[string]string{
				"foo.%": UnknownVariableValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"hello":       cty.NumberIntVal(12),
					"hello.world": cty.NumberIntVal(10),
				}),
			}),
			map[string]string{
				"foo.%":           "2",
				"foo.hello":       "12",
				"foo.hello.world": "10",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("world"),
				}),
			}),
			map[string]string{
				"foo.#": "2",
				"foo.0": "hello",
				"foo.1": "world",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Set(cty.Number)),
			}),
			map[string]string{
				"foo.#": UnknownVariableValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("hello"),
						"baz": cty.StringVal("world"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bloo"),
						"baz": cty.StringVal("blaa"),
					}),
				}),
			}),
			map[string]string{
				"foo.#":     "2",
				"foo.0.bar": "hello",
				"foo.0.baz": "world",
				"foo.1.bar": "bloo",
				"foo.1.baz": "blaa",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("hello"),
						"baz": cty.ListVal([]cty.Value{
							cty.True,
							cty.True,
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("bloo"),
						"baz": cty.ListVal([]cty.Value{
							cty.False,
							cty.True,
						}),
					}),
				}),
			}),
			map[string]string{
				"foo.#":       "2",
				"foo.0.bar":   "hello",
				"foo.0.baz.#": "2",
				"foo.0.baz.0": "true",
				"foo.0.baz.1": "true",
				"foo.1.bar":   "bloo",
				"foo.1.baz.#": "2",
				"foo.1.baz.0": "false",
				"foo.1.baz.1": "true",
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.UnknownVal(cty.Object(map[string]cty.Type{
						"bar": cty.String,
						"baz": cty.List(cty.Bool),
						"bap": cty.Map(cty.Number),
					})),
				}),
			}),
			map[string]string{
				"foo.#":       "1",
				"foo.0.bar":   UnknownVariableValue,
				"foo.0.baz.#": UnknownVariableValue,
				"foo.0.bap.%": UnknownVariableValue,
			},
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				})),
			})),
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Value.GoString(), func(t *testing.T) {
			got := FlatmapValueFromHCL2(test.Value)

			for _, problem := range deep.Equal(got, test.Want) {
				t.Error(problem)
			}
		})
	}
}

func TestFlatmapValueFromHCL2FromFlatmap(t *testing.T) {
	tests := []struct {
		Name string
		Map  map[string]string
		Type cty.Type
	}{
		{
			"empty flatmap with collections",
			map[string]string{},
			cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.String),
				"bar": cty.Set(cty.String),
			}),
		},
		{
			"nil flatmap with collections",
			nil,
			cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.String),
				"bar": cty.Set(cty.String),
			}),
		},
		{
			"empty flatmap with nested collections",
			map[string]string{},
			cty.Object(map[string]cty.Type{
				"foo": cty.Object(
					map[string]cty.Type{
						"baz": cty.Map(cty.String),
					},
				),
				"bar": cty.Set(cty.String),
			}),
		},
		{
			"partial flatmap with nested collections",
			map[string]string{
				"foo.baz.%":   "1",
				"foo.baz.key": "val",
			},
			cty.Object(map[string]cty.Type{
				"foo": cty.Object(
					map[string]cty.Type{
						"baz": cty.Map(cty.String),
						"biz": cty.Map(cty.String),
					},
				),
				"bar": cty.Set(cty.String),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			val, err := HCL2ValueFromFlatmap(test.Map, test.Type)
			if err != nil {
				t.Fatal(err)
			}

			got := FlatmapValueFromHCL2(val)

			for _, problem := range deep.Equal(got, test.Map) {
				t.Error(problem)
			}
		})
	}
}
func TestHCL2ValueFromFlatmap(t *testing.T) {
	tests := []struct {
		Flatmap map[string]string
		Type    cty.Type
		Want    cty.Value
		WantErr string
	}{
		{
			Flatmap: map[string]string{},
			Type:    cty.EmptyObject,
			Want:    cty.EmptyObjectVal,
		},
		{
			Flatmap: map[string]string{
				"ignored": "foo",
			},
			Type: cty.EmptyObject,
			Want: cty.EmptyObjectVal,
		},
		{
			Flatmap: map[string]string{
				"foo": "blah",
				"bar": "true",
				"baz": "12.5",
				"unk": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.Bool,
				"baz": cty.Number,
				"unk": cty.Bool,
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("blah"),
				"bar": cty.True,
				"baz": cty.NumberFloatVal(12.5),
				"unk": cty.UnknownVal(cty.Bool),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "0",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(cty.String),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.List(cty.String)),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "1",
				"foo.0": "hello",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "2",
				"foo.0": "true",
				"foo.1": "false",
				"foo.2": "ignored", // (because the count is 2, so this is out of range)
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.Bool),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.True,
					cty.False,
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "2",
				"foo.0": "hello",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Tuple([]cty.Type{
					cty.String,
					cty.Bool,
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.TupleVal([]cty.Value{
					cty.StringVal("hello"),
					cty.NullVal(cty.Bool),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Tuple([]cty.Type{
					cty.String,
					cty.Bool,
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Tuple([]cty.Type{
					cty.String,
					cty.Bool,
				})),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "0",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetValEmpty(cty.String),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Set(cty.String)),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#":        "1",
				"foo.24534534": "hello",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.StringVal("hello"),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#":        "1",
				"foo.24534534": "true",
				"foo.95645644": "true",
				"foo.34533452": "false",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Bool),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.True,
					cty.False,
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.%": "0",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapValEmpty(cty.String),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.%":       "2",
				"foo.baz":     "true",
				"foo.bar.baz": "false",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.Bool),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"baz":     cty.True,
					"bar.baz": cty.False,
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.%": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Map(cty.Bool),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.Map(cty.Bool)),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#":     "2",
				"foo.0.bar": "hello",
				"foo.0.baz": "1",
				"foo.1.bar": "world",
				"foo.1.baz": "false",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.Bool,
				})),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("hello"),
						"baz": cty.True,
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("world"),
						"baz": cty.False,
					}),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#":            "2",
				"foo.34534534.bar": "hello",
				"foo.34534534.baz": "1",
				"foo.93453345.bar": "world",
				"foo.93453345.baz": "false",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.Bool,
				})),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("hello"),
						"baz": cty.True,
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("world"),
						"baz": cty.False,
					}),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"foo.#": "not-valid",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.String),
			}),
			WantErr: `invalid count value for "foo." in state: strconv.Atoi: parsing "not-valid": invalid syntax`,
		},
		{
			Flatmap: nil,
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				})),
			}),
			Want: cty.NullVal(cty.Object(map[string]cty.Type{
				"foo": cty.Set(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				})),
			})),
		},
		{
			Flatmap: map[string]string{
				"foo.#":   "2",
				"foo.0.%": "2",
				"foo.0.a": "a",
				"foo.0.b": "b",
				"foo.1.%": "1",
				"foo.1.a": "a",
			},
			Type: cty.Object(map[string]cty.Type{
				"foo": cty.List(cty.Map(cty.String)),
			}),

			Want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"a": cty.StringVal("a"),
						"b": cty.StringVal("b"),
					}),
					cty.MapVal(map[string]cty.Value{
						"a": cty.StringVal("a"),
					}),
				}),
			}),
		},
		{
			Flatmap: map[string]string{
				"single.#":                 "1",
				"single.~1.value":          "a",
				"single.~1.optional":       UnknownVariableValue,
				"two.#":                    "2",
				"two.~2381914684.value":    "a",
				"two.~2381914684.optional": UnknownVariableValue,
				"two.~2798940671.value":    "b",
				"two.~2798940671.optional": UnknownVariableValue,
			},
			Type: cty.Object(map[string]cty.Type{
				"single": cty.Set(
					cty.Object(map[string]cty.Type{
						"value":    cty.String,
						"optional": cty.String,
					}),
				),
				"two": cty.Set(
					cty.Object(map[string]cty.Type{
						"optional": cty.String,
						"value":    cty.String,
					}),
				),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"single": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"value":    cty.StringVal("a"),
						"optional": cty.UnknownVal(cty.String),
					}),
				}),
				"two": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"value":    cty.StringVal("a"),
						"optional": cty.UnknownVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"value":    cty.StringVal("b"),
						"optional": cty.UnknownVal(cty.String),
					}),
				}),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v as %#v", test.Flatmap, test.Type), func(t *testing.T) {
			got, err := HCL2ValueFromFlatmap(test.Flatmap, test.Type)

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("succeeded; want error: %s", test.WantErr)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				if got == cty.NilVal {
					t.Fatalf("result is cty.NilVal; want valid placeholder value")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
