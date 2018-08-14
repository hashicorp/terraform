package command

import (
	"fmt"
	"strings"

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

	cmdFlags := c.Meta.flagSet("state pull")
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

	// Get the state
	env := c.Workspace()
	state, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}
	if err := state.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	s := state.State()
	if s == nil {
		// Output on "error" so it shows up on stderr
		c.Ui.Error("Empty state (no state)")

		return 0
	}

	c.Ui.Error("state pull not yet updated for new state types")
	return 1

	/*
		var buf bytes.Buffer
		if err := terraform.WriteState(s, &buf); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		c.Ui.Output(buf.String())
	*/

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
