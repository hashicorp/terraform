package flatmap

import (
	"reflect"
	"testing"
)

func TestMapDelete(t *testing.T) {
	m := Flatten(map[string]interface{}{
		"foo": "bar",
		"routes": []map[string]string{
			map[string]string{
				"foo": "bar",
			},
		},
	})

	m.Delete("routes")

	expected := Map(map[string]string{"foo": "bar"})
	if !reflect.DeepEqual(m, expected) {
		t.Fatalf("bad: %#v", m)
	}
}

func TestMapKeys(t *testing.T) {
	cases := []struct {
		Input  map[string]string
		Output []string
	}{
		{
			Input: map[string]string{
				"foo":       "bar",
				"bar.#":     "bar",
				"bar.0.foo": "bar",
				"bar.0.baz": "bar",
			},
			Output: []string{
				"foo",
				"bar",
			},
		},
	}

	for _, tc := range cases {
		actual := Map(tc.Input).Keys()
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("input: %#v\n\nbad: %#v", tc.Input, actual)
		}
	}
}

func TestMapMerge(t *testing.T) {
	cases := []struct {
		One    map[string]string
		Two    map[string]string
		Result map[string]string
	}{
		{
			One: map[string]string{
				"foo": "bar",
				"bar": "nope",
			},
			Two: map[string]string{
				"bar": "baz",
				"baz": "buz",
			},
			Result: map[string]string{
				"foo": "bar",
				"bar": "baz",
				"baz": "buz",
			},
		},
	}

	for i, tc := range cases {
		Map(tc.One).Merge(Map(tc.Two))
		if !reflect.DeepEqual(tc.One, tc.Result) {
			t.Fatalf("case %d bad: %#v", i, tc.One)
		}
	}
}
