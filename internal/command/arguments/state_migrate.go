// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

const lockFileName = ".terraform.lock.hcl"

// StateMigrate represents the command-line arguments for the state migrate command.
type StateMigrate struct {
	SourceLockFilePath      string
	DestinationLockFilePath string
	Upgrade                 bool
	InputEnabled            bool

	ViewType ViewType
}

// ParseStateMigrate processes CLI arguments, returning a StateMigrate value and
// diagnostics. If errors are encountered, a StateMigrate value is still returned
// representing the best effort interpretation of the arguments.
func ParseStateMigrate(args []string) (*StateMigrate, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	migrate := &StateMigrate{
		ViewType: ViewHuman,
	}

	var srcLockFilePath, dstLockFilePath string
	var upgrade, inputEnabled bool
	cmdFlags := defaultFlagSet("state migrate")
	cmdFlags.StringVar(&srcLockFilePath, "source-provider-lock-file", "", "Path to a provider lock file for the source provider.")
	cmdFlags.StringVar(&dstLockFilePath, "destination-provider-lock-file", "", "Path to a provider lock file for the destination provider.")
	cmdFlags.BoolVar(&upgrade, "upgrade", false, "Trigger upgrade of the provider.")
	cmdFlags.BoolVar(&inputEnabled, "input", true, "Enable input for interactive prompts.")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
		return migrate, diags
	}

	migrate.Upgrade = upgrade
	migrate.InputEnabled = inputEnabled

	if inputEnabled {
		// lock file paths are only to be used in automation
		if srcLockFilePath != "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Conflicting command-line flags provided",
				"-source-provider-lock-file cannot be used outside of automation (with -input=true)",
			))
		}
		if dstLockFilePath != "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Conflicting command-line flags provided",
				"-destination-provider-lock-file cannot be used outside of automation (with -input=true)",
			))
		}
		if len(diags) > 0 {
			return migrate, diags
		}

	}
	if dstLockFilePath == "" {
		// setting default here instead of in the flag definition
		// to make check above free of side effects
		dstLockFilePath = lockFileName
	}

	if srcLockFilePath != "" {
		srcFilename := filepath.Base(srcLockFilePath)
		if srcFilename != lockFileName {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid source-provider-lock-file",
				fmt.Sprintf("Expected lock file name to be %s, got: %s", lockFileName, srcFilename),
			))
		} else {
			migrate.SourceLockFilePath = srcLockFilePath
		}
	}

	dstFilename := filepath.Base(dstLockFilePath)
	if dstFilename != lockFileName {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid destination-provider-lock-file",
			fmt.Sprintf("Expected lock file name to be %s, got: %s", lockFileName, dstFilename),
		))
	} else {
		migrate.DestinationLockFilePath = dstLockFilePath
	}

	return migrate, diags
}
