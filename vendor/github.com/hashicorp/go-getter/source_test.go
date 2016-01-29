package getter

import (
	"testing"
)

func TestSourceDirSubdir(t *testing.T) {
	cases := []struct {
		Input    string
		Dir, Sub string
	}{
		{
			"hashicorp.com",
			"hashicorp.com", "",
		},
		{
			"hashicorp.com//foo",
			"hashicorp.com", "foo",
		},
		{
			"hashicorp.com//foo?bar=baz",
			"hashicorp.com?bar=baz", "foo",
		},
		{
			"file://foo//bar",
			"file://foo", "bar",
		},
	}

	for i, tc := range cases {
		adir, asub := SourceDirSubdir(tc.Input)
		if adir != tc.Dir {
			t.Fatalf("%d: bad dir: %#v", i, adir)
		}
		if asub != tc.Sub {
			t.Fatalf("%d: bad sub: %#v", i, asub)
		}
	}
}
