package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/remote"
)

type PushCommand struct {
	Meta
}

func (c *PushCommand) Run(args []string) int {
	var force bool
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("push", flag.ContinueOnError)
	cmdFlags.BoolVar(&force, "force", false, "")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Check for a remote state file
	local, _, err := remote.ReadLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	if local == nil || local.Remote == nil {
		c.Ui.Error("Remote state not enabled!")
		return 1
	}

	// Attempt to push the state
	change, err := remote.PushState(local.Remote, force)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to push state: %v", err))
		return 1
	}

	// Use an error exit code if the update was not a success
	if !change.SuccessfulPush() {
		c.Ui.Error(fmt.Sprintf("%s", change))
		return 1
	} else {
		c.Ui.Output(fmt.Sprintf("%s", change))
	}
	return 0
}

func (c *PushCommand) Help() string {
	helpText := `
Usage: terraform push [options]

  Uploads the latest state to the remote server.

Options:

  -force                 Forces the upload of the local state, ignoring any
                         conflicts. This should be used carefully, as force pushing
						 can cause remote state information to be lost.

`
	return strings.TrimSpace(helpText)
}

func (c *PushCommand) Synopsis() string {
	return "Uploads the the local state to the remote server"
}
