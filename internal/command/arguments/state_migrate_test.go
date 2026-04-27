// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateMigrateArgs(t *testing.T) {
	testCases := []struct {
		rawArgs       []string
		expectedArgs  *StateMigrate
		expectedDiags tfdiags.Diagnostics
	}{
		{
			rawArgs: []string{""},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      "",
				DestinationLockFilePath: ".terraform.lock.hcl",
				Upgrade:                 false,
				InputEnabled:            true,
				ViewType:                ViewHuman,
			},
		},
		{ // set or override all flags
			rawArgs: []string{
				"-source-provider-lock-file", "/some/path/.terraform.lock.hcl",
				"-destination-provider-lock-file", "/some/other/path/.terraform.lock.hcl",
				"-upgrade",
				"-input=false",
			},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      "/some/path/.terraform.lock.hcl",
				DestinationLockFilePath: "/some/other/path/.terraform.lock.hcl",
				Upgrade:                 true,
				InputEnabled:            false,
				ViewType:                ViewHuman,
			},
		},
		{
			rawArgs: []string{"-input=false", "-source-provider-lock-file", "foo"},
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
					"Expected lock file name to be .terraform.lock.hcl, got: foo",
				),
			},
		},
		{
			rawArgs: []string{"-input=false", "-destination-provider-lock-file", "foo"},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      "",
				DestinationLockFilePath: "",
				Upgrade:                 false,
				InputEnabled:            false,
				ViewType:                ViewHuman,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid destination-provider-lock-file",
					"Expected lock file name to be .terraform.lock.hcl, got: foo",
				),
			},
		},
		{ // set lock file paths outside of automation
			rawArgs: []string{
				"-source-provider-lock-file", "/src/.terraform.lock.hcl",
				"-destination-provider-lock-file", "/dst/.terraform.lock.hcl",
			},
			expectedArgs: &StateMigrate{
				SourceLockFilePath:      "",
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
			},
		},
	}

	for i, tc := range testCases {
		args, diags := ParseStateMigrate(tc.rawArgs)
		tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectedDiags)
		if diff := cmp.Diff(tc.expectedArgs, args); diff != "" {
			t.Fatalf("%d: supplied: %q, got unexpected arguments:\n%s", i, tc.rawArgs, diff)
		}
	}
}
