// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateMigrateArgs_valid(t *testing.T) {
	t.Parallel()

	// Create lock files to use in test, as validation will check for the existence of the
	// file if a path is supplied via CLI flags.
	td := t.TempDir()
	userLockFilePath := filepath.Join(td, ".terraform.lock.hcl")
	if err := os.WriteFile(userLockFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test lock file: %s", err)
	}

	invalidLockFilePath := filepath.Join(td, "invalid.lock.hcl")
	if err := os.WriteFile(invalidLockFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test lock file: %s", err)
	}

	testCases := []struct {
		rawArgs      []string
		expectedArgs *StateMigrate
	}{
		{
			rawArgs: []string{""},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      ".terraform.lock.hcl",
				DestinationLockFilePath: ".terraform.lock.hcl",
				Upgrade:                 false,
				InputEnabled:            true,
				ViewType:                ViewHuman,
			},
		},
		{ // set or override all flags
			rawArgs: []string{
				"-source-provider-lock-file", userLockFilePath,
				"-destination-provider-lock-file", userLockFilePath,
				"-upgrade",
				"-json",
				"-input=false",
			},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      userLockFilePath,
				DestinationLockFilePath: userLockFilePath,
				Upgrade:                 true,
				InputEnabled:            false,
				ViewType:                ViewJSON,
			},
		},
	}

	for i, tc := range testCases {
		args, diags := ParseStateMigrate(tc.rawArgs)
		if diags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if diff := cmp.Diff(tc.expectedArgs, args); diff != "" {
			t.Fatalf("%d: supplied: %q, got unexpected arguments:\n%s", i, tc.rawArgs, diff)
		}
	}
}

func TestStateMigrateArgs_invalid(t *testing.T) {
	t.Parallel()

	// Create lock files to use in test, as validation will check for the existence of the
	// file if a path is supplied via CLI flags.
	td := t.TempDir()
	userLockFilePath := filepath.Join(td, ".terraform.lock.hcl")
	if err := os.WriteFile(userLockFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test lock file: %s", err)
	}

	invalidLockFilePath := filepath.Join(td, "invalid.lock.hcl")
	if err := os.WriteFile(invalidLockFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test lock file: %s", err)
	}

	testCases := []struct {
		rawArgs       []string
		expectedArgs  *StateMigrate
		expectedDiags tfdiags.Diagnostics
	}{
		{
			rawArgs: []string{"-input=false", "-source-provider-lock-file", invalidLockFilePath},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      "",
				DestinationLockFilePath: ".terraform.lock.hcl",
				Upgrade:                 false,
				InputEnabled:            false,
				ViewType:                ViewHuman,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid source-provider-lock-file",
					"Expected lock file name to be .terraform.lock.hcl, got: invalid.lock.hcl",
				),
			},
		},
		{
			rawArgs: []string{"-input=false", "-destination-provider-lock-file", invalidLockFilePath},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      ".terraform.lock.hcl",
				DestinationLockFilePath: "",
				Upgrade:                 false,
				InputEnabled:            false,
				ViewType:                ViewHuman,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid destination-provider-lock-file",
					"Expected lock file name to be .terraform.lock.hcl, got: invalid.lock.hcl",
				),
			},
		},
		{ // set lock file paths outside of automation
			rawArgs: []string{
				"-source-provider-lock-file", userLockFilePath,
				"-destination-provider-lock-file", invalidLockFilePath,
			},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      userLockFilePath,
				DestinationLockFilePath: "",
				Upgrade:                 false,
				InputEnabled:            true,
				ViewType:                ViewHuman,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Conflicting command-line flags provided",
					"-source-provider-lock-file cannot be used outside of automation (with -input=true)",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Conflicting command-line flags provided",
					"-destination-provider-lock-file cannot be used outside of automation (with -input=true)",
				),
				// Diagnostic about invalid lock file regardless of this flag not being allow
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid destination-provider-lock-file",
					"Expected lock file name to be .terraform.lock.hcl, got: invalid.lock.hcl",
				),
			},
		},
		{ // JSON output outside of automation
			rawArgs: []string{
				"-json",
			},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      ".terraform.lock.hcl",
				DestinationLockFilePath: ".terraform.lock.hcl",
				Upgrade:                 false,
				InputEnabled:            true,
				ViewType:                ViewJSON,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Conflicting command-line flags provided",
					"-json cannot be used outside of automation (with -input=true)",
				),
			},
		},
	}

	for i, tc := range testCases {
		args, diags := ParseStateMigrate(tc.rawArgs)

		if !diags.HasErrors() {
			t.Fatalf("expected diagnostics, but got none")
		}
		tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectedDiags)

		if diff := cmp.Diff(tc.expectedArgs, args); diff != "" {
			t.Fatalf("%d: supplied: %q, got unexpected arguments:\n%s", i, tc.rawArgs, diff)
		}
	}
}
