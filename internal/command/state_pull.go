package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// StatePullCommand is a Command implementation that shows a single resource.
type StatePullCommand struct {
	Meta
	StateMeta
}

func (c *StatePullCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("state pull")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteBackendVersionConflict(b)

	// Get the state manager for the current workspace
	env, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
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
Usage: terraform [global options] state pull [options]

  Pull the state from its location, upgrade the local copy, and output it
  to stdout.

  This command "pulls" the current state and outputs it to stdout.
  As part of this process, Terraform will upgrade the state format of the
  local copy to the current version.

  The primary use of this is for state stored remotely. This command
  will still work with local state but is less useful for this.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePullCommand) Synopsis() string {
	return "Pull current state and output to stdout"
}
