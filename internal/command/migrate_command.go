// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

// MigrateCommand is a Command implementation that shows help for
// the migrate subcommands.
type MigrateCommand struct {
	Meta
}

func (c *MigrateCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *MigrateCommand) Help() string {
	helpText := `
Usage: terraform [global options] migrate <subcommand> [options] [args]

  This command has subcommands for running source code migrations.

  Migrations transform your Terraform configuration files to accommodate
  breaking changes introduced by provider upgrades or Terraform Core updates.

`
	return strings.TrimSpace(helpText)
}

func (c *MigrateCommand) Synopsis() string {
	return "Run source code migrations"
}
