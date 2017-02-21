package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

// DebugCommand is a Command implementation that just shows help for
// the subcommands nested below it.
type DebugCommand struct {
	Meta
}

func (c *DebugCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *DebugCommand) Help() string {
	helpText := `
Usage: terraform debug <subcommand> [options] [args]

  This command has subcommands for debug output management
`
	return strings.TrimSpace(helpText)
}

func (c *DebugCommand) Synopsis() string {
	return "Debug output management (experimental)"
}
