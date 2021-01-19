package main

import (
	"strings"

	"github.com/mitchellh/cli"
)

// graphCommand implements the "stresstest graph" command, which is the
// container for various subcommands related to graph testing.
type graphCommand struct {
}

var _ cli.Command = (*graphCommand)(nil)

func (c *graphCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *graphCommand) Synopsis() string {
	return "Stress-test the graph build and walk"
}

func (c *graphCommand) Help() string {
	return strings.TrimSpace(`
Usage: stresstest graph [subcommand]

...
`)
}
