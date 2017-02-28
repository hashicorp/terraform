package command

import (
	"strings"

	"github.com/mitchellh/cli"
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
Usage: terraform state <subcommand> [options] [args]

  This command has subcommands for advanced state management.

  These subcommands can be used to slice and dice the Terraform state.
  This is sometimes necessary in advanced cases. For your safety, all
  state management commands that modify the state create a timestamped
  backup of the state prior to making modifications.

  The structure and output of the commands is specifically tailored to work
  well with the common Unix utilities such as grep, awk, etc. We recommend
  using those tools to perform more advanced state tasks.

`
	return strings.TrimSpace(helpText)
}

func (c *StateCommand) Synopsis() string {
	return "Advanced state management"
}
