// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "testing"

func TestValidWorkspaceName(t *testing.T) {
	cases := map[string]struct {
		input string
		valid bool
	}{
		"foobar": {
			input: "foobar",
			valid: true,
		},
		"valid symbols": {
			input: "-._~@:",
			valid: true,
		},
		"includes space": {
			input: "two words",
			valid: false,
		},
		"empty string": {
			input: "",
			valid: false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			valid := ValidWorkspaceName(tc.input)
			if valid != tc.valid {
				t.Fatalf("unexpected output when processing input %q. Wanted %v got %v", tc.input, tc.valid, valid)
			}
		})
	}
}
