package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/remote"
)

type PullCommand struct {
	Meta
}

func (c *PullCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("pull", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Recover the local state if any
	local, _, err := remote.ReadLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	if local == nil || local.Remote == nil {
		c.Ui.Error("Remote state not enabled!")
		return 1
	}

	// Attempt the state refresh
	change, err := remote.RefreshState(local.Remote)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Failed to refresh from remote state: %v", err))
		return 1
	}

	// Use an error exit code if the update was not a success
	if !change.SuccessfulPull() {
		c.Ui.Error(fmt.Sprintf("%s", change))
		return 1
	} else {
		c.Ui.Output(fmt.Sprintf("%s", change))
	}
	return 0
}

func (c *PullCommand) Help() string {
	helpText := `
Usage: terraform pull [options]

  Refreshes the cached state file from the remote server.

`
	return strings.TrimSpace(helpText)
}

func (c *PullCommand) Synopsis() string {
	return "Refreshes the local state copy from the remote server"
}
