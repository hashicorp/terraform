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
  This is sometimes necessary in advanced cases.

  For unusual use-cases, you can also optionally create additional named
  states associated with your current configuration, but we recommend
  considering that to be a last resort. In most cases, each configuration
  directory should have only one state.
`
	// FIXME: Ideally we'd also be able to customize the presentation of the
	// subcommands to have two different categories here -- separating the
	// manipulation of the current state from managing multiple states -- but
	// the mitchellh/cli package seems to want to generate that part of the
	// output itself, so not clear how we could customize it.
	return strings.TrimSpace(helpText)
}

func (c *StateCommand) Synopsis() string {
	return "Advanced state management"
}
