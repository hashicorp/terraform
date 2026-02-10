// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"reflect"
	"testing"

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
				TestDirectory: "tests",
			},
		},
		"fs-mirror": {
			[]string{"-fs-mirror=/path/to/mirror"},
			&ProvidersLock{
				FSMirrorDir:   "/path/to/mirror",
				TestDirectory: "tests",
			},
		},
		"net-mirror": {
			[]string{"-net-mirror=https://mirror.example.com/"},
			&ProvidersLock{
				NetMirrorURL:  "https://mirror.example.com/",
				TestDirectory: "tests",
			},
		},
		"single platform": {
			[]string{"-platform=linux_amd64"},
			&ProvidersLock{
				Platforms:     FlagStringSlice{"linux_amd64"},
				TestDirectory: "tests",
			},
		},
		"multiple platforms": {
			[]string{"-platform=linux_amd64", "-platform=darwin_arm64"},
			&ProvidersLock{
				Platforms:     FlagStringSlice{"linux_amd64", "darwin_arm64"},
				TestDirectory: "tests",
			},
		},
		"enable-plugin-cache": {
			[]string{"-enable-plugin-cache"},
			&ProvidersLock{
				EnablePluginCache: true,
				TestDirectory:     "tests",
			},
		},
		"test-directory": {
			[]string{"-test-directory=mytests"},
			&ProvidersLock{
				TestDirectory: "mytests",
			},
		},
		"provider arguments": {
			[]string{"hashicorp/aws", "hashicorp/random"},
			&ProvidersLock{
				Providers:     []string{"hashicorp/aws", "hashicorp/random"},
				TestDirectory: "tests",
			},
		},
		"all options": {
			[]string{"-fs-mirror=/mirror", "-platform=linux_amd64", "-enable-plugin-cache", "-test-directory=mytests", "hashicorp/aws"},
			&ProvidersLock{
				FSMirrorDir:       "/mirror",
				Platforms:         FlagStringSlice{"linux_amd64"},
				EnablePluginCache: true,
				TestDirectory:     "mytests",
				Providers:         []string{"hashicorp/aws"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersLock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
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
		"unknown flag": {
			[]string{"-unknown"},
			&ProvidersLock{
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
			},
		},
		"mirror collision": {
			[]string{"-fs-mirror=/foo/", "-net-mirror=https://example.com/"},
			&ProvidersLock{
				FSMirrorDir:   "/foo/",
				NetMirrorURL:  "https://example.com/",
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid installation method options",
					"The -fs-mirror and -net-mirror command line options are mutually-exclusive.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersLock(tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}
