package command

import (
	"strings"

	"github.com/posener/complete"
)

type WorkspaceShowCommand struct {
	Meta
}

func (c *WorkspaceShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("workspace show")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	workspace := c.Workspace()
	c.Ui.Output(workspace)

	return 0
}

func (c *WorkspaceShowCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *WorkspaceShowCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceShowCommand) Help() string {
	helpText := `
Usage: terraform workspace show

  Show the name of the current workspace.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceShowCommand) Synopsis() string {
	return "Show the name of the current workspace"
}
