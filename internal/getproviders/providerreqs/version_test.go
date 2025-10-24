// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providerreqs

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
			"> 1.0.0, >= 1.0.0, <= 1.0.0, < 1.0.0, != 1.0.0",
		},
		"multiple": {
			MustParseVersionConstraints(">= 3.0, < 4.0"),
			">= 3.0.0, < 4.0.0",
		},
		"duplicates removed": {
			MustParseVersionConstraints(">= 1.2.3, 1.2.3, ~> 1.2, 1.2.3"),
			"~> 1.2, >= 1.2.3, 1.2.3",
		},
		"equivalent duplicates removed": {
			MustParseVersionConstraints(">= 2.68, >= 2.68.0"),
			">= 2.68.0",
		},
		"consistent ordering, exhaustive": {
			// This weird jumble is just to exercise the different sort
			// ordering codepaths. Hopefully nothing quite this horrific
			// shows up often in practice.
			MustParseVersionConstraints("< 1.2.3, <= 1.2.3, != 1.2.3, 1.2.3+local.2, 1.2.3+local.1, = 1.2.4, = 1.2.3, > 2, > 1.2.3, >= 1.2.3, ~> 1.2.3, ~> 1.2"),
			"~> 1.2, > 1.2.3, >= 1.2.3, 1.2.3, ~> 1.2.3, <= 1.2.3, < 1.2.3, != 1.2.3, 1.2.3+local.1, 1.2.3+local.2, 1.2.4, > 2.0.0",
		},
		"consistent ordering, more typical": {
			// This one is aiming to simulate a common situation where
			// various different modules express compatible constraints
			// but some modules are more constrained than others. The
			// combined results here can be kinda confusing, but hopefully
			// ordering them consistently makes them a _little_ easier to read.
			MustParseVersionConstraints("~> 1.2, >= 1.2, 1.2.4"),
			">= 1.2.0, ~> 1.2, 1.2.4",
		},
		"consistent ordering, disjoint": {
			// One situation where our presentation of version constraints is
			// particularly important is when a user accidentally ends up with
			// disjoint constraints that can therefore never match. In that
			// case, our ordering should hopefully make it easier to determine
			// that the constraints are disjoint, as a first step to debugging,
			// by showing > or >= constrains sorted after < or <= constraints.
			MustParseVersionConstraints(">= 2, >= 1.2, < 1.3"),
			">= 1.2.0, < 1.3.0, >= 2.0.0",
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
