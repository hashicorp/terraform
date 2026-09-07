// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
				Vars:           &Vars{},
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
				Vars:              &Vars{},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersLock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
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
				Vars:           &Vars{},
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
				Vars:           &Vars{},
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
				Vars:           &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersLock(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseProvidersLock_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersLock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}
