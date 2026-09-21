// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

// StateCommand is a Command implementation that just shows help for
// the subcommands nested below it.
type StateCommand struct {
	StateMeta
}

func (c *StateCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *StateCommand) Help() string {
	helpText := `
Usage: terraform [global options] state <subcommand> [options] [args]

  This command has subcommands for advanced state management.

  These subcommands can be used to slice and dice the Terraform state.
  This is sometimes necessary in advanced cases. For your safety, state
  management commands that modify a local state file create a timestamped
  backup of the state prior to making modifications.

`
	return strings.TrimSpace(helpText)
}

func (c *StateCommand) Synopsis() string {
	return "Advanced state management"
}
