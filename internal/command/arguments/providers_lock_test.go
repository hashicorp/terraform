// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProvidersLock_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *ProvidersLock
	}{
		"defaults": {
			nil,
			&ProvidersLock{
				TestsDirectory: "tests",
			},
		},
		"all options": {
			[]string{
				"-platform=linux_amd64",
				"-platform=darwin_arm64",
				"-fs-mirror=mirror",
				"-test-directory=integration-tests",
				"-enable-plugin-cache",
				"hashicorp/test",
			},
			&ProvidersLock{
				Platforms:         FlagStringSlice{"linux_amd64", "darwin_arm64"},
				FSMirrorDir:       "mirror",
				TestsDirectory:    "integration-tests",
				EnablePluginCache: true,
				Providers:         []string{"hashicorp/test"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersLock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseProvidersLock_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *ProvidersLock
		wantDiags tfdiags.Diagnostics
	}{
		"mirror collision": {
			[]string{
				"-fs-mirror=foo",
				"-net-mirror=https://example.com",
			},
			&ProvidersLock{
				FSMirrorDir:    "foo",
				NetMirrorURL:   "https://example.com",
				TestsDirectory: "tests",
				Providers:      []string{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid installation method options",
					"The -fs-mirror and -net-mirror command line options are mutually-exclusive.",
				),
			},
		},
		"unknown flag": {
			[]string{"-wat"},
			&ProvidersLock{
				TestsDirectory: "tests",
				Providers:      []string{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
			},
		},
		"unknown flag and mirror collision": {
			[]string{
				"-wat",
				"-fs-mirror=foo",
				"-net-mirror=https://example.com",
			},
			&ProvidersLock{
				TestsDirectory: "tests",
				Providers:      []string{"-fs-mirror=foo", "-net-mirror=https://example.com"},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersLock(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}
