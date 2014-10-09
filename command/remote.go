package command

import "strings"

// RemoteCommand is a Command implementation that is used to
// enable and disable remote state management
type RemoteCommand struct {
	Meta
}

func (c *RemoteCommand) Run(args []string) int {
	return 0
}

func (c *RemoteCommand) Help() string {
	helpText := `
Usage: terraform remote [options]

  Configures Terraform to use a remote state server. This allows state
  to be pulled down when necessary and then pushed to the server when
  updated. In this mode, the state file does not need to be stored durably
  since the remote server provides the durability.

Options:

  -remote=name           Name of the state file in the state storage server.
                         Optional, default does not use remote storage.

  -remote-auth=token     Authentication token for state storage server.
                         Optional, defaults to blank.

  -remote-server=url     URL of the remote storage server.

`
	return strings.TrimSpace(helpText)
}

func (c *RemoteCommand) Synopsis() string {
	return "Configures remote state management"
}
