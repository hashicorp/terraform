// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
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
	dir := c.Meta.WorkingDir.RootModuleDir()
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

	// TODO: Account for cases where lock entries are missing

	migrateOpts := &backendMigrateOpts{
		ViewType: args.ViewType,
	}

	// Load the source backend
	var source string
	if smi.Backend != nil {
		source = fmt.Sprintf("backend %q", smi.Backend.Type)

		srcB, _, srcDiags := c.Meta.backendInitFromConfig(smi.Backend)
		diags = diags.Append(srcDiags)
		if !diags.HasErrors() {
			migrateOpts.SourceType = smi.Backend.Type
			migrateOpts.Source = srcB
		}
	} else if smi.StateStore != nil {
		source = fmt.Sprintf("state store %q (%s)", smi.StateStore.Type,
			smi.StateStore.ProviderAddr.ForDisplay())

		srcLocks, srcLockDiags := depsfile.LoadLocksFromFile(args.SourceLockFilePath)
		if srcLockDiags.HasErrors() {
			diags = diags.Append(srcLockDiags)
			stateMigrate.Diagnostics(diags)
			return 1
		} else {
			diags = diags.Append(srcLockDiags)

			srcB, _, _, srcDiags := c.Meta.stateStoreInitFromConfig(smi.StateStore, srcLocks)
			diags = diags.Append(srcDiags)
			if !diags.HasErrors() {
				migrateOpts.SourceType = smi.StateStore.Type
				migrateOpts.Source = srcB
			}
		}
	}

	// Load the destination backend
	rootMod := cfg.Module
	var destination string
	if rootMod.Backend != nil {
		destination = fmt.Sprintf("backend %q", rootMod.Backend.Type)

		dstB, _, dstDiags := c.Meta.backendInitFromConfig(rootMod.Backend)
		diags = diags.Append(dstDiags)
		if !diags.HasErrors() {
			migrateOpts.DestinationType = rootMod.Backend.Type
			migrateOpts.Destination = dstB
		}
	} else if rootMod.StateStore != nil {
		destination = fmt.Sprintf("state store %q (%s)", rootMod.StateStore.Type,
			rootMod.StateStore.ProviderAddr.ForDisplay())

		dstLocks, dstLockDiags := depsfile.LoadLocksFromFile(args.DestinationLockFilePath)
		if dstLockDiags.HasErrors() {
			diags = diags.Append(dstLockDiags)
			stateMigrate.Diagnostics(diags)
			return 1
		} else {
			diags = diags.Append(dstLockDiags)

			dstB, _, _, dstDiags := c.Meta.stateStoreInitFromConfig(rootMod.StateStore, dstLocks)
			diags = diags.Append(dstDiags)
			if !diags.HasErrors() {
				migrateOpts.DestinationType = rootMod.StateStore.Type
				migrateOpts.Destination = dstB
			}
		}
	} else {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unknown migration destination",
			"No configuration was provided for where to migrate the state to. Please ensure that a file with a .tf extension is present and contains valid state_store or backend configuration inside the terraform block.",
		))
	}

	// present all errors from above together so user can fix them all at once
	if diags.HasErrors() {
		stateMigrate.Diagnostics(diags)
		return 1
	}

	stateMigrate.Log("Migrating state from %s to %s...", source, destination)

	// Perform the migration from source to destination
	err := c.Meta.backendMigrateState(migrateOpts)
	if err != nil {
		diags = diags.Append(fmt.Errorf("migration failed: %w", err))
		stateMigrate.Diagnostics(diags)
		return 1
	}

	stateMigrate.Diagnostics(diags)

	stateMigrate.Log("Finished migrating state from %s to %s...", source, destination)

	return 0
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
