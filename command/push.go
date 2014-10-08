package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
)

type PushCommand struct {
	Meta
}

func (c *PushCommand) Run(args []string) int {
	var force bool
	var remoteConf terraform.RemoteState
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("push", flag.ContinueOnError)
	cmdFlags.StringVar(&remoteConf.Name, "remote", "", "")
	cmdFlags.StringVar(&remoteConf.Server, "remote-server", "", "")
	cmdFlags.StringVar(&remoteConf.AuthToken, "remote-auth", "", "")
	cmdFlags.BoolVar(&force, "force", false, "")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Validate the remote configuration if given
	var conf *terraform.RemoteState
	if !remoteConf.Empty() {
		if err := remote.ValidateConfig(&remoteConf); err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}
		conf = &remoteConf
	} else {
		// Recover the local state if any
		local, _, err := remote.ReadLocalState()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}
		if local == nil || local.Remote == nil {
			c.Ui.Error("No remote state server configured")
			return 1
		}
		conf = local.Remote
	}

	// Attempt to push the state
	change, err := remote.PushState(conf, force)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Failed to push state: %v", err))
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

  Uploads the local state file to the remote server. This is done automatically
  by commands when remote state if configured, but can also be done manually
  using this command.

Options:

  -force                 Forces the upload of the local state, ignoring any
                         conflicts. This should be used carefully, as force pushing
						 can cause remote state information to be lost.

  -remote=name           Name of the state file in the state storage server.
                         Optional, default does not use remote storage.

  -remote-auth=token     Authentication token for state storage server.
                         Optional, defaults to blank.

  -remote-server=url     URL of the remote storage server.

`
	return strings.TrimSpace(helpText)
}

func (c *PushCommand) Synopsis() string {
	return "Uploads the the local state to the remote server"
}
