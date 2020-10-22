package getproviders

import (
	"testing"
)

func TestVersionConstraintsString(t *testing.T) {
	testCases := map[string]struct {
		spec VersionConstraints
		want string
	}{
		"exact": {
			MustParseVersionConstraints("1.2.3"),
			"1.2.3",
		},
		"prerelease": {
			MustParseVersionConstraints("1.2.3-beta"),
			"1.2.3-beta",
		},
		"metadata": {
			MustParseVersionConstraints("1.2.3+foo.bar"),
			"1.2.3+foo.bar",
		},
		"prerelease and metadata": {
			MustParseVersionConstraints("1.2.3-beta+foo.bar"),
			"1.2.3-beta+foo.bar",
		},
		"major only": {
			MustParseVersionConstraints(">= 3"),
			">= 3.0.0",
		},
		"major only with pessimistic operator": {
			MustParseVersionConstraints("~> 3"),
			"~> 3.0",
		},
		"pessimistic minor": {
			MustParseVersionConstraints("~> 3.0"),
			"~> 3.0",
		},
		"pessimistic patch": {
			MustParseVersionConstraints("~> 3.0.0"),
			"~> 3.0.0",
		},
		"other operators": {
			MustParseVersionConstraints("> 1.0.0, < 1.0.0, >= 1.0.0, <= 1.0.0, != 1.0.0"),
			"> 1.0.0, < 1.0.0, >= 1.0.0, <= 1.0.0, != 1.0.0",
		},
		"multiple": {
			MustParseVersionConstraints(">= 3.0, < 4.0"),
			">= 3.0.0, < 4.0.0",
		},
		"duplicates removed": {
			MustParseVersionConstraints(">= 1.2.3, 1.2.3, ~> 1.2, 1.2.3"),
			">= 1.2.3, 1.2.3, ~> 1.2",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := VersionConstraintsString(tc.spec)

			if got != tc.want {
				t.Errorf("wrong\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
