package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestInterpolateFuncConcat(t *testing.T) {
	cases := []struct {
		Args   []string
		Result string
		Error  bool
	}{
		{
			[]string{"foo", "bar", "baz"},
			"foobarbaz",
			false,
		},

		{
			[]string{"foo", "bar"},
			"foobar",
			false,
		},

		{
			[]string{"foo"},
			"foo",
			false,
		},
	}

	for i, tc := range cases {
		actual, err := interpolationFuncConcat(nil, tc.Args...)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if actual != tc.Result {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}

func TestInterpolateFuncFile(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	path := tf.Name()
	tf.Write([]byte("foo"))
	tf.Close()
	defer os.Remove(path)

	cases := []struct {
		Args   []string
		Result string
		Error  bool
	}{
		{
			[]string{path},
			"foo",
			false,
		},

		// Invalid path
		{
			[]string{"/i/dont/exist"},
			"",
			true,
		},

		// Too many args
		{
			[]string{"foo", "bar"},
			"",
			true,
		},
	}

	for i, tc := range cases {
		actual, err := interpolationFuncFile(nil, tc.Args...)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if actual != tc.Result {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}

func TestInterpolateFuncJoin(t *testing.T) {
	cases := []struct {
		Args   []string
		Result string
		Error  bool
	}{
		{
			[]string{","},
			"",
			true,
		},

		{
			[]string{",", "foo"},
			"foo",
			false,
		},

		{
			[]string{",", "foo", "bar"},
			"foo,bar",
			false,
		},

		{
			[]string{
				".",
				fmt.Sprintf(
					"foo%sbar%sbaz",
					InterpSplitDelim,
					InterpSplitDelim),
			},
			"foo.bar.baz",
			false,
		},
	}

	for i, tc := range cases {
		actual, err := interpolationFuncJoin(nil, tc.Args...)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if actual != tc.Result {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}

func TestInterpolateFuncLookup(t *testing.T) {
	cases := []struct {
		M      map[string]string
		Args   []string
		Result string
		Error  bool
	}{
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "bar"},
			"baz",
			false,
		},

		// Invalid key
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "baz"},
			"",
			true,
		},

		// Too many args
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "bar", "baz"},
			"",
			true,
		},
	}

	for i, tc := range cases {
		actual, err := interpolationFuncLookup(tc.M, tc.Args...)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if actual != tc.Result {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}
