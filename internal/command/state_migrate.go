// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
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

	if _, err := os.Stat(args.SourceLockFilePath); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unreadable source provider lock file",
			err.Error(),
		))
	}

	// TODO: Is there a reason to do migration without a lock file?
	if _, err := os.Stat(args.DestinationLockFilePath); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unreadable destination provider lock file",
			err.Error(),
		))
	}

	// TODO: implement
	// stateMigrate.Log("migrating from %s to %s", "source", "destination")

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

  -source-provider-lock-file	   Path to a provider lock file for the source provider (requires -input=false).

  -destination-provider-lock-file  Path to a provider lock file for the destination provider (requires -input=false).

  -upgrade  					   Trigger upgrade of the provider.
  
  -input=true					   Enable input for interactive prompts (defaults to true, set to false in automation).
`
	return strings.TrimSpace(helpText)
}

func (c *StateMigrateCommand) Synopsis() string {
	return "Migrate the state from one location to another"
}
