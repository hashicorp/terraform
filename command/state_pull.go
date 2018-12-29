package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/mitchellh/cli"
)

// StatePullCommand is a Command implementation that shows a single resource.
type StatePullCommand struct {
	Meta
	StateMeta
}

func (c *StatePullCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("state pull")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// Get the state manager for the current workspace
	env := c.Workspace()
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh state: %s", err))
		return 1
	}

	// Get a statefile object representing the latest snapshot
	stateFile := statemgr.Export(stateMgr)

	if stateFile != nil { // we produce no output if the statefile is nil
		var buf bytes.Buffer
		err = statefile.Write(stateFile, &buf)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to write state: %s", err))
			return 1
		}

		c.Ui.Output(buf.String())
	}

	return 0
}

func (c *StatePullCommand) Help() string {
	helpText := `
Usage: terraform state pull [options]

  Pull the state from its location and output it to stdout.

  This command "pulls" the current state and outputs it to stdout.
  The primary use of this is for state stored remotely. This command
  will still work with local state but is less useful for this.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePullCommand) Synopsis() string {
	return "Pull current state and output to stdout"
}
