package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestDefaults(t *testing.T) {
	tests := []struct {
		Input, Defaults cty.Value
		Want            cty.Value
		WantErr         string
	}{
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hey"),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hey"),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			WantErr: `.a: target type does not expect an attribute named "a"`,
		},

		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.NullVal(cty.String),
					cty.StringVal("hey"),
					cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("hey"),
					cty.StringVal("hello"),
				}),
			}),
		},
		{
			// Using defaults with single set elements is a pretty
			// odd thing to do, but this behavior is just here because
			// it generalizes from how we handle collections. It's
			// tested only to ensure it doesn't change accidentally
			// in future.
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.NullVal(cty.String),
					cty.StringVal("hey"),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.StringVal("hey"),
					cty.StringVal("hello"),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"x": cty.NullVal(cty.String),
					"y": cty.StringVal("hey"),
					"z": cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"x": cty.StringVal("hello"),
					"y": cty.StringVal("hey"),
					"z": cty.StringVal("hello"),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hello"),
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
				}),
			}),
		},
		{
			Input: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hey"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.NullVal(cty.String),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hey"),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"b": cty.StringVal("hello"),
			}),
			Want: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hey"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hello"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hey"),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hello"),
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("boop"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.NullVal(cty.String),
					}),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hello"),
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.SetVal([]cty.Value{
					// After applying defaults, the one with a null value
					// coalesced with the one with a non-null value,
					// and so there's only one left.
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"boop": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
					"beep": cty.ObjectVal(map[string]cty.Value{
						"b": cty.NullVal(cty.String),
					}),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.StringVal("hello"),
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"boop": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
					"beep": cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hello"),
					}),
				}),
			}),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"b": cty.StringVal("hey"),
					}),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			WantErr: `.a: the default value for a collection of an object type must itself be an object type, not string`,
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.NullVal(cty.String),
					cty.StringVal("hey"),
					cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				// The default value for a list must be a single value
				// of the list's element type which provides defaults
				// for each element separately, so the default for a
				// list of string should be just a single string, not
				// a list of string.
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
				}),
			}),
			WantErr: `.a: invalid default value for string: string required`,
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.NullVal(cty.String),
					cty.StringVal("hey"),
					cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
			}),
			WantErr: `.a: the default value for a tuple type must itself be a tuple type, not string`,
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.NullVal(cty.String),
					cty.StringVal("hey"),
					cty.NullVal(cty.String),
				}),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.StringVal("hello 0"),
					cty.StringVal("hello 1"),
					cty.StringVal("hello 2"),
				}),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.TupleVal([]cty.Value{
					cty.StringVal("hello 0"),
					cty.StringVal("hey"),
					cty.StringVal("hello 2"),
				}),
			}),
		},
		{
			// There's no reason to use this function for plain primitive
			// types, because the "default" argument in a variable definition
			// already has the equivalent behavior. This function is only
			// to deal with the situation of a complex-typed variable where
			// only parts of the data structure are optional.
			Input:    cty.NullVal(cty.String),
			Defaults: cty.StringVal("hello"),
			WantErr:  `only object types and collections of object types can have defaults applied`,
		},
		// When applying default values to collection types, null collections in the
		// input should result in empty collections in the output.
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.List(cty.String)),
				"b": cty.NullVal(cty.Map(cty.String)),
				"c": cty.NullVal(cty.Set(cty.String)),
			}),
			Defaults: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("hello"),
				"b": cty.StringVal("hi"),
				"c": cty.StringVal("greetings"),
			}),
			Want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListValEmpty(cty.String),
				"b": cty.MapValEmpty(cty.String),
				"c": cty.SetValEmpty(cty.String),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("defaults(%#v, %#v)", test.Input, test.Defaults), func(t *testing.T) {
			got, gotErr := Defaults(test.Input, test.Defaults)

			if test.WantErr != "" {
				if gotErr == nil {
					t.Fatalf("unexpected success\nwant error: %s", test.WantErr)
				}
				if got, want := gotErr.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if gotErr != nil {
				t.Fatalf("unexpected error\ngot:  %s", gotErr.Error())
			}

			if !test.Want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
