package versionlock

import (
	"testing"
)

func TestGetLockedCLIVersion(t *testing.T) {
	tests := []struct {
		Dir  string
		Want string
	}{
		{"testdata", ""},
		{"testdata/a", "1.0.0"},
		{"testdata/a/b", "2.0.0"},
		{"testdata/a/b/c", "2.0.0"},
		{"testdata/a/invalid", ""},
	}

	for _, test := range tests {
		t.Run(test.Dir, func(t *testing.T) {
			var got string
			gotV, _ := GetLockedCLIVersion(test.Dir)
			if gotV != nil {
				got = gotV.String()
			}

			if got != test.Want {
				t.Errorf("wrong result\ndir:  %s\ngot:  %s\nwant: %s", test.Dir, got, test.Want)
			}
		})
	}
}
