package format

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestObjectValueIDOrName(t *testing.T) {
	tests := []struct {
		obj    cty.Value
		id     [2]string
		name   [2]string
		either [2]string
	}{
		{
			cty.NullVal(cty.EmptyObject),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.UnknownVal(cty.EmptyObject),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.EmptyObjectVal,
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo-123"),
			}),
			[...]string{"id", "foo-123"},
			[...]string{"", ""},
			[...]string{"id", "foo-123"},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("foo-123"),
				"name": cty.StringVal("awesome-foo"),
			}),
			[...]string{"id", "foo-123"},
			[...]string{"name", "awesome-foo"},
			[...]string{"id", "foo-123"},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("awesome-foo"),
			}),
			[...]string{"name", "awesome-foo"},
			[...]string{"name", "awesome-foo"},
			[...]string{"name", "awesome-foo"},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("awesome-foo"),
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("My Awesome Foo"),
				}),
			}),
			[...]string{"name", "awesome-foo"},
			[...]string{"name", "awesome-foo"},
			[...]string{"name", "awesome-foo"},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("My Awesome Foo"),
					"name": cty.StringVal("awesome-foo"),
				}),
			}),
			[...]string{"", ""},
			[...]string{"tags.name", "awesome-foo"},
			[...]string{"tags.name", "awesome-foo"},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.StringVal("My Awesome Foo"),
				}),
			}),
			[...]string{"", ""},
			[...]string{"tags.Name", "My Awesome Foo"},
			[...]string{"tags.Name", "My Awesome Foo"},
		},

		// The following are degenerate cases, included to make sure we don't
		// crash when we encounter them. If you're here fixing a reported panic
		// in this formatter, this is the place to add a new test case.
		{
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.True,
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.StringVal("foo"),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.NullVal(cty.Map(cty.String)),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.UnknownVal(cty.Map(cty.String)),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.True,
				}),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.UnknownVal(cty.String),
				}),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{
					"Name": cty.NullVal(cty.String),
				}),
			}),
			[...]string{"", ""},
			[...]string{"", ""},
			[...]string{"", ""},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.obj), func(t *testing.T) {
			obj := test.obj
			gotIDKey, gotIDVal := ObjectValueID(obj)
			gotNameKey, gotNameVal := ObjectValueName(obj)
			gotEitherKey, gotEitherVal := ObjectValueIDOrName(obj)

			if got, want := [...]string{gotIDKey, gotIDVal}, test.id; got != want {
				t.Errorf("wrong ObjectValueID result\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := [...]string{gotNameKey, gotNameVal}, test.name; got != want {
				t.Errorf("wrong ObjectValueName result\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := [...]string{gotEitherKey, gotEitherVal}, test.either; got != want {
				t.Errorf("wrong ObjectValueIDOrName result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}
