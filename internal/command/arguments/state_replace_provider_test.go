// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateReplaceProvider_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateReplaceProvider
	}{
		"provider addresses only": {
			[]string{"hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				AutoApprove:         false,
				BackupPath:          "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				IgnoreRemoteVersion: false,
				FromProviderAddr:    "hashicorp/aws",
				ToProviderAddr:      "acmecorp/aws",
			},
		},
		"auto approve": {
			[]string{"-auto-approve", "hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				AutoApprove:         true,
				BackupPath:          "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				IgnoreRemoteVersion: false,
				FromProviderAddr:    "hashicorp/aws",
				ToProviderAddr:      "acmecorp/aws",
			},
		},
		"all options": {
			[]string{
				"-auto-approve",
				"-backup=backup.tfstate",
				"-lock=false",
				"-lock-timeout=5s",
				"-state=state.tfstate",
				"-ignore-remote-version",
				"hashicorp/aws",
				"acmecorp/aws",
			},
			&StateReplaceProvider{
				AutoApprove:         true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    5 * time.Second,
				StatePath:           "state.tfstate",
				IgnoreRemoteVersion: true,
				FromProviderAddr:    "hashicorp/aws",
				ToProviderAddr:      "acmecorp/aws",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateReplaceProvider(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateReplaceProvider_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateReplaceProvider
		wantDiags tfdiags.Diagnostics
	}{
		"no arguments": {
			nil,
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the from and to provider addresses.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b", "c", "d"},
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the from and to provider addresses.",
				),
			},
		},
		"unknown flag": {
			[]string{"-invalid", "hashicorp/google", "acmecorp/google"},
			&StateReplaceProvider{
				BackupPath:       "-",
				StateLock:        true,
				FromProviderAddr: "hashicorp/google",
				ToProviderAddr:   "acmecorp/google",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -invalid",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateReplaceProvider(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}
