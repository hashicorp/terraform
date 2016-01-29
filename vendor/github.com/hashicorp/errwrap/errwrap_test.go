package errwrap

import (
	"fmt"
	"testing"
)

func TestWrappedError_impl(t *testing.T) {
	var _ error = new(wrappedError)
}

func TestGetAll(t *testing.T) {
	cases := []struct {
		Err error
		Msg string
		Len int
	}{
		{},
		{
			fmt.Errorf("foo"),
			"foo",
			1,
		},
		{
			fmt.Errorf("bar"),
			"foo",
			0,
		},
		{
			Wrapf("bar", fmt.Errorf("foo")),
			"foo",
			1,
		},
		{
			Wrapf("{{err}}", fmt.Errorf("foo")),
			"foo",
			2,
		},
		{
			Wrapf("bar", Wrapf("baz", fmt.Errorf("foo"))),
			"foo",
			1,
		},
	}

	for i, tc := range cases {
		actual := GetAll(tc.Err, tc.Msg)
		if len(actual) != tc.Len {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
		for _, v := range actual {
			if v.Error() != tc.Msg {
				t.Fatalf("%d: bad: %#v", i, actual)
			}
		}
	}
}

func TestGetAllType(t *testing.T) {
	cases := []struct {
		Err  error
		Type interface{}
		Len  int
	}{
		{},
		{
			fmt.Errorf("foo"),
			"foo",
			0,
		},
		{
			fmt.Errorf("bar"),
			fmt.Errorf("foo"),
			1,
		},
		{
			Wrapf("bar", fmt.Errorf("foo")),
			fmt.Errorf("baz"),
			2,
		},
		{
			Wrapf("bar", Wrapf("baz", fmt.Errorf("foo"))),
			Wrapf("", nil),
			0,
		},
	}

	for i, tc := range cases {
		actual := GetAllType(tc.Err, tc.Type)
		if len(actual) != tc.Len {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}
