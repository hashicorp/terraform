// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MigrateList represents the command-line arguments for the
// "terraform migrate list" command.
type MigrateList struct {
	Detail   bool
	ViewType ViewType
}

// ParseMigrateList parses command-line flags for "terraform migrate list".
func ParseMigrateList(args []string) (*MigrateList, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonOutput bool

	migrateList := &MigrateList{}

	cmdFlags := defaultFlagSet("migrate list")
	cmdFlags.BoolVar(&migrateList.Detail, "detail", false, "detail")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected no positional arguments",
		))
	}

	switch {
	case jsonOutput:
		migrateList.ViewType = ViewJSON
	default:
		migrateList.ViewType = ViewHuman
	}

	return migrateList, diags
}

// MigrateApply represents the command-line arguments for the
// "terraform migrate" command.
type MigrateApply struct {
	MigrationID string
	DryRun      bool
	Step        bool
	ViewType    ViewType
}

// ParseMigrateApply parses command-line flags for "terraform migrate".
// The migration ID (containing /) can appear before or after flags.
func ParseMigrateApply(args []string) (*MigrateApply, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonOutput bool

	migrateApply := &MigrateApply{}

	// Extract the migration ID from args before flag parsing, since Go's
	// flag package stops at the first non-flag argument.
	var flagArgs []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && strings.Contains(arg, "/") {
			if migrateApply.MigrationID != "" {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Too many migration IDs",
					"Expected exactly one migration ID argument.",
				))
			}
			migrateApply.MigrationID = arg
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}

	cmdFlags := defaultFlagSet("migrate")
	cmdFlags.BoolVar(&migrateApply.DryRun, "dry-run", false, "dry-run")
	cmdFlags.BoolVar(&migrateApply.Step, "step", false, "step")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

	if err := cmdFlags.Parse(flagArgs); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if remaining := cmdFlags.Args(); len(remaining) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unexpected arguments",
			"Unexpected positional arguments after flags.",
		))
	}

	if migrateApply.MigrationID == "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Missing migration ID",
			"Usage: terraform migrate <namespace/provider/name> [-dry-run] [-step]",
		))
	} else {
		parts := strings.SplitN(migrateApply.MigrationID, "/", 4)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid migration ID format",
				"Migration ID must be in the format namespace/provider/name, with all parts non-empty",
			))
		}
	}

	if migrateApply.DryRun && migrateApply.Step {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Incompatible command line flags",
			"The -dry-run and -step flags are mutually exclusive",
		))
	}

	switch {
	case jsonOutput:
		migrateApply.ViewType = ViewJSON
	default:
		migrateApply.ViewType = ViewHuman
	}

	return migrateApply, diags
}
