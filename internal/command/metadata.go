package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

// MetadataCommand is a Command implementation that just shows help for
// the subcommands nested below it.
type MetadataCommand struct {
}

func (c *MetadataCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *MetadataCommand) Help() string {
	helpText := `
Usage: terraform metadata <subcommand>

  This command has subcommands for displaying metadata about this Terraform
  version.

`
	return strings.TrimSpace(helpText)
}

func (c *MetadataCommand) Synopsis() string {
	return "Terraform metadata"
}
