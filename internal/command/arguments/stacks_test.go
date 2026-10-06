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
				Args:                   []string{},
			},
		},
		"any additional flags and arguments are stored": {
			[]string{"-plugin-cache-dir=foobar1", "-fizzbuzz", "foobar2", "foobar3"},
			&Stacks{
				PluginCacheDirOverride: "foobar1",
				Args:                   []string{"-fizzbuzz", "foobar2", "foobar3"},
			},
		},
		"if plugin-cache-dir flag supplied twice, second one takes precedence": {
			[]string{"-plugin-cache-dir=foobar1", "-plugin-cache-dir=foobar2", "foobar3"},
			&Stacks{
				PluginCacheDirOverride: "foobar2",
				Args:                   []string{"foobar3"},
			},
		},
		"if plugin-cache-dir flag supplied after positional arguments it is removed": {
			// This is due to how we remove the plugin-cache-dir flag but preserve other flags.
			// If downstream code in the future needs to handle plugin-cache-dir flags differently,
			// this behavior will need to be revisited.
			[]string{"-plugin-cache-dir=foobar1", "foobar2", "-plugin-cache-dir=foobar3"},
			&Stacks{
				PluginCacheDirOverride: "foobar1",
				Args:                   []string{"foobar2"},
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
