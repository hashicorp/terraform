// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

// MigrateCommand is a Command implementation that either shows help for
// the migrate subcommands or delegates to MigrateApplyCommand when the
// first argument looks like a migration ID (contains "/").
type MigrateCommand struct {
	Meta
}

func (c *MigrateCommand) Run(args []string) int {
	// If the first arg looks like a migration ID (contains /), delegate
	// directly to the apply command for convenience.
	if len(args) > 0 && strings.Contains(args[0], "/") {
		apply := &MigrateApplyCommand{Meta: c.Meta}
		return apply.Run(args)
	}

	return cli.RunResultHelp
}

func (c *MigrateCommand) Help() string {
	helpText := `
Usage: terraform [global options] migrate <subcommand> [options] [args]

  This command has subcommands for running source code migrations.

  Migrations transform your Terraform configuration files to accommodate
  breaking changes introduced by provider upgrades. Available subcommands
  include:

    list     List available migrations for the current working directory
    <id>     Apply a specific migration (e.g. hashicorp/aws/v3-to-v4)

`
	return strings.TrimSpace(helpText)
}

func (c *MigrateCommand) Synopsis() string {
	return "Run source code migrations"
}
