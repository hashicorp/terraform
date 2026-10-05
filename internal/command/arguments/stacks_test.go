// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseStacks_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Stacks
	}{
		"defaults": {
			nil,
			&Stacks{
				Args: []string{},
			},
		},
		"plugin-cache-dir path": {
			[]string{"-plugin-cache-dir=foobar"},
			&Stacks{
				PluginCacheDirOverride: "foobar",
				Args:           []string{},
			},
		},
		"additional positional arguments are stored": {
			[]string{"-plugin-cache-dir=foobar1", "foobar2", "foobar3"},
			&Stacks{
				PluginCacheDirOverride: "foobar1",
				Args:           []string{"foobar2", "foobar3"},
			},
		},
		"unrecognized flags are ignored": {
			[]string{"-unrecognized"},
			&Stacks{
				Args: []string{},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStacks(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

// There are no invalid scenarios currently.
