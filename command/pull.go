package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
)

type PullCommand struct {
	Meta
}

func (c *PullCommand) Run(args []string) int {
	var remoteConf terraform.RemoteState
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("pull", flag.ContinueOnError)
	cmdFlags.StringVar(&remoteConf.Name, "remote", "", "")
	cmdFlags.StringVar(&remoteConf.Server, "remote-server", "", "")
	cmdFlags.StringVar(&remoteConf.AuthToken, "remote-auth", "", "")
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

	// Attempt the state refresh
	change, err := remote.RefreshState(conf)
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

  Refreshes the cached state file from the remote server. It can also
  be used to perform the initial clone of the state file and setup the
  remote server configuration to use remote state storage.

Options:

  -remote=name           Name of the state file in the state storage server.
                         Optional, default does not use remote storage.

  -remote-auth=token     Authentication token for state storage server.
                         Optional, defaults to blank.

  -remote-server=url     URL of the remote storage server.

`
	return strings.TrimSpace(helpText)
}

func (c *PullCommand) Synopsis() string {
	return "Refreshes the local state copy from the remote server"
}
