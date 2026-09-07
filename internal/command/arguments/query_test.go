// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseQuery_policies(t *testing.T) {
	testCases := map[string]struct {
		args         []string
		wantPolicies []string
	}{
		"no flag omitted": {
			args:         nil,
			wantPolicies: nil,
		},
		"single path equals syntax": {
			args:         []string{"-policies=/some/path"},
			wantPolicies: []string{"/some/path"},
		},
		"single path space syntax": {
			args:         []string{"-policies", "/some/path"},
			wantPolicies: []string{"/some/path"},
		},
		"multiple paths repeated flag": {
			args:         []string{"-policies=/path/one", "-policies=/path/two"},
			wantPolicies: []string{"/path/one", "/path/two"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseQuery(tc.args)
			if diags.HasErrors() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.wantPolicies, got.PolicyPaths); diff != "" {
				t.Errorf("unexpected PolicyPaths\n%s", diff)
			}
		})
	}
}
