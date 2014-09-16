package module

import (
	"testing"
)

func TestFileDetector(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{"./foo", "file:///pwd/foo"},
		{"./foo?foo=bar", "file:///pwd/foo?foo=bar"},
		{"foo", "file:///pwd/foo"},
		{"/foo", "file:///foo"},
		{"/foo?bar=baz", "file:///foo?bar=baz"},
	}

	pwd := "/pwd"
	f := new(FileDetector)
	for i, tc := range cases {
		output, ok, err := f.Detect(tc.Input, pwd)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if !ok {
			t.Fatal("not ok")
		}

		if output != tc.Output {
			t.Fatalf("%d: bad: %#v", i, output)
		}
	}
}

func TestFileDetector_noPwd(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
		Err    bool
	}{
		{"./foo", "", true},
		{"foo", "", true},
		{"/foo", "file:///foo", false},
	}

	pwd := ""
	f := new(FileDetector)
	for i, tc := range cases {
		output, ok, err := f.Detect(tc.Input, pwd)
		if (err != nil) != tc.Err {
			t.Fatalf("%d: err: %s", i, err)
		}
		if !ok {
			t.Fatal("not ok")
		}

		if output != tc.Output {
			t.Fatalf("%d: bad: %#v", i, output)
		}
	}
}
