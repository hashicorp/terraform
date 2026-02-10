// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateReplaceProvider_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateReplaceProvider
	}{
		"defaults with positionals": {
			[]string{"hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				AutoApprove: false,
				BackupPath:  "-",
				StateLock:   true,
				StatePath:   "",
				From:        "hashicorp/aws",
				To:          "acmecorp/aws",
			},
		},
		"auto-approve": {
			[]string{"-auto-approve", "hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				AutoApprove: true,
				BackupPath:  "-",
				StateLock:   true,
				From:        "hashicorp/aws",
				To:          "acmecorp/aws",
			},
		},
		"all flags": {
			[]string{
				"-auto-approve",
				"-backup=backup.tfstate",
				"-lock=false",
				"-lock-timeout=10s",
				"-state=state.tfstate",
				"-ignore-remote-version",
				"hashicorp/aws",
				"acmecorp/aws",
			},
			&StateReplaceProvider{
				AutoApprove:         true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    10_000_000_000,
				StatePath:           "state.tfstate",
				IgnoreRemoteVersion: true,
				From:                "hashicorp/aws",
				To:                  "acmecorp/aws",
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
		"unknown flag": {
			[]string{"-unknown", "from", "to"},
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
				From:       "from",
				To:         "to",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
			},
		},
		"no arguments": {
			nil,
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state replace-provider command requires a from and to provider FQN.",
				),
			},
		},
		"too few arguments": {
			[]string{"from"},
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
				From:       "from",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state replace-provider command requires a from and to provider FQN.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b", "c"},
			&StateReplaceProvider{
				BackupPath: "-",
				StateLock:  true,
				From:       "a",
				To:         "b",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state replace-provider command requires a from and to provider FQN.",
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
