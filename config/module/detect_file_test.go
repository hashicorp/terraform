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
		{"foo", "file:///pwd/foo"},
		{"/foo", "file:///foo"},
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
