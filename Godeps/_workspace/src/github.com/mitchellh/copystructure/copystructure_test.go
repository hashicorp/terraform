package copystructure

import (
	"reflect"
	"testing"
)

func TestCopy_complex(t *testing.T) {
	v := map[string]interface{}{
		"foo": []string{"a", "b"},
		"bar": "baz",
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_primitive(t *testing.T) {
	cases := []interface{}{
		42,
		"foo",
		1.2,
	}

	for _, tc := range cases {
		result, err := Copy(tc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if result != tc {
			t.Fatalf("bad: %#v", result)
		}
	}
}

func TestCopy_map(t *testing.T) {
	v := map[string]interface{}{
		"bar": "baz",
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_slice(t *testing.T) {
	v := []string{"bar", "baz"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}
