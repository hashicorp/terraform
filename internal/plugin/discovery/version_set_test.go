package discovery

import (
	"fmt"
	"testing"
)

func TestVersionSet(t *testing.T) {
	tests := []struct {
		ConstraintStr string
		VersionStr    string
		ShouldHave    bool
	}{
		// These test cases are not exhaustive since the underlying go-version
		// library is well-tested. This is mainly here just to exercise our
		// wrapper code, but also used as an opportunity to cover some basic
		// but important cases such as the ~> constraint so that we'll be more
		// likely to catch any accidental breaking behavior changes in the
		// underlying library.
		{
			">=1.0.0",
			"1.0.0",
			true,
		},
		{
			">=1.0.0",
			"0.0.0",
			false,
		},
		{
			">=1.0.0",
			"1.1.0-beta1",
			false,
		},
		{
			">=1.0.0",
			"1.1.0",
			true,
		},
		{
			"~>1.1.0-a",
			"1.1.0-beta1",
			true,
		},
		{
			"~>1.1.0",
			"1.1.2",
			true,
		},
		{
			"~>1.1.0",
			"1.2.0",
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s has %s", test.ConstraintStr, test.VersionStr), func(t *testing.T) {
			accepted, err := ConstraintStr(test.ConstraintStr).Parse()
			if err != nil {
				t.Fatalf("unwanted error parsing constraints string %q: %s", test.ConstraintStr, err)
			}

			version, err := VersionStr(test.VersionStr).Parse()
			if err != nil {
				t.Fatalf("unwanted error parsing version string %q: %s", test.VersionStr, err)
			}

			if got, want := accepted.Allows(version), test.ShouldHave; got != want {
				t.Errorf("Has returned %#v; want %#v", got, want)
			}
		})
	}
}
