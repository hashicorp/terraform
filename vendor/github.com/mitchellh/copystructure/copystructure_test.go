package copystructure

import (
	"reflect"
	"testing"
	"time"
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

func TestCopy_primitivePtr(t *testing.T) {
	cases := []interface{}{
		42,
		"foo",
		1.2,
	}

	for _, tc := range cases {
		result, err := Copy(&tc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(result, &tc) {
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

func TestCopy_struct(t *testing.T) {
	type test struct {
		Value string
	}

	v := test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structPtr(t *testing.T) {
	type test struct {
		Value string
	}

	v := &test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structNil(t *testing.T) {
	type test struct {
		Value string
	}

	var v *test
	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if v, ok := result.(*test); !ok {
		t.Fatalf("bad: %#v", result)
	} else if v != nil {
		t.Fatalf("bad: %#v", v)
	}
}

func TestCopy_structNested(t *testing.T) {
	type TestInner struct{}

	type Test struct {
		Test *TestInner
	}

	v := Test{}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structUnexported(t *testing.T) {
	type test struct {
		Value string

		private string
	}

	v := test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_time(t *testing.T) {
	type test struct {
		Value time.Time
	}

	v := test{Value: time.Now().UTC()}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}
