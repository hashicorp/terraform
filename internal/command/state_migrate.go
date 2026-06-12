// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateMigrateCommand is a Command implementation that migrates
// the state file from one location to another
type StateMigrateCommand struct {
	Meta
}

func (c *StateMigrateCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.Meta.View.Configure(common)

	args, diags := arguments.ParseStateMigrate(rawArgs)

	stateMigrate := views.NewStateMigrate(args.ViewType, c.View)

	if diags.HasErrors() {
		stateMigrate.Diagnostics(diags)
		return 1
	}

	c.Meta.input = args.InputEnabled

	if args.SourceLockFilePath != "" {
		if _, err := os.Stat(args.SourceLockFilePath); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unreadable source provider lock file",
				fmt.Sprintf("%q: %s", args.SourceLockFilePath, err.Error()),
			))
		}
	}

	dir := c.Meta.WorkingDir.RootModuleDir()

	// It is valid for the destination lockfile to be missing
	// while state exists - e.g. through the use of builtin provider
	// or outputs and use of a builtin backend
	// (as opposed to pluggable state store).
	if args.DestinationLockFilePath != "" {
		if _, err := os.Stat(args.DestinationLockFilePath); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unreadable destination provider lock file",
				fmt.Sprintf("%q: %s", args.DestinationLockFilePath, err.Error()),
			))
		}
	}

	// return validation errors early if there are any
	if diags.HasErrors() {
		stateMigrate.Diagnostics(diags)
		return 1
	}

	c.Meta.includeStateMigrateFiles = true
	cfg, mDiags := c.Meta.loadConfig(dir)
	if mDiags.HasErrors() {
		diags = diags.Append(mDiags)
		stateMigrate.Diagnostics(diags)
		return 1
	}

	smi := cfg.Module.StateMigrationInstructions
	if smi == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No state migration instructions found",
			"No instructions were found in the configuration files. Please ensure that a file with a .tfmigrate.hcl extension is present and contains valid state migration instructions.",
		))
		stateMigrate.Diagnostics(diags)
		return 1
	}

	var source string
	if smi.Backend != nil {
		source = fmt.Sprintf("backend %q", smi.Backend.Type)
	} else if smi.StateStore != nil {
		source = fmt.Sprintf("state store %q (%s)", smi.StateStore.Type,
			smi.StateStore.ProviderAddr)
	}

	rootMod := cfg.Module
	var destination string
	if rootMod.Backend != nil {
		destination = fmt.Sprintf("backend %q", rootMod.Backend.Type)
	} else if rootMod.StateStore != nil {
		destination = fmt.Sprintf("state store %q (%s)", rootMod.StateStore.Type,
			rootMod.StateStore.ProviderAddr)
	} else {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unknown migration destination",
			"No configuration was provided for where to migrate the state to. Please ensure that a file with a .tf extension is present and contains valid state_store or backend configuration inside the terraform block.",
		))
		stateMigrate.Diagnostics(diags)
		return 1
	}

	stateMigrate.Log("Migrating state from %s to %s...", source, destination)

	// TODO: Load the source backend
	// TODO: Load the destination backend
	// TODO: Perform the migration from source to destination

	diags = diags.Append(errors.New("Not implemented yet"))
	stateMigrate.Diagnostics(diags)
	return 1
}

func (c *StateMigrateCommand) Help() string {
	helpText := `
Usage: terraform [global options] state migrate [options]

  Migrate state from source declared in the migration configuration (*.tfmigrate.hcl)
  to the destination declared in the root module (*.tf).

  An error will be returned if the migration fails, e.g. if the state
  is inaccessible or the migration configuration is invalid.

Options:

  -source-provider-lock-file       Path to a provider lock file for the source provider (requires -input=false).
                                   Defaults to using the working directory's .terraform.lock.hcl file.

  -destination-provider-lock-file  Path to a provider lock file for the destination provider (requires -input=false).
                                   Defaults to using the working directory's .terraform.lock.hcl file.

  -upgrade                         Trigger upgrade of the provider used for state storage.

  -input=true                      Enable input for interactive prompts (defaults to true, set to false in automation).
`
	return strings.TrimSpace(helpText)
}

func (c *StateMigrateCommand) Synopsis() string {
	return "Migrate the state from one location to another"
}
