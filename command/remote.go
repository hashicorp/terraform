package command

import (
	"strings"
)

type RemoteCommand struct {
	Meta
}

func (c *RemoteCommand) Run(argsRaw []string) int {
	// Duplicate the args so we can munge them without affecting
	// future subcommand invocations which will do the same.
	args := make([]string, len(argsRaw))
	copy(args, argsRaw)
	args = c.Meta.process(args, false)

	if len(args) == 0 {
		c.Ui.Error(c.Help())
		return 1
	}

	switch args[0] {
	case "config":
		cmd := &RemoteConfigCommand{Meta: c.Meta}
		return cmd.Run(args[1:])
	case "pull":
		cmd := &RemotePullCommand{Meta: c.Meta}
		return cmd.Run(args[1:])
	case "push":
		cmd := &RemotePushCommand{Meta: c.Meta}
		return cmd.Run(args[1:])
	default:
		c.Ui.Error(c.Help())
		return 1
	}
}

func (c *RemoteCommand) Help() string {
	helpText := `
Usage: terraform remote <subcommand> [options]

  Configure remote state storage with Terraform.

Options:

  -no-color   If specified, output won't contain any color.

Available subcommands:

  config      Configure the remote storage settings.
  pull        Sync the remote storage by downloading to local storage.
  push        Sync the remote storage by uploading the local storage.

`
	return strings.TrimSpace(helpText)
}

func (c *RemoteCommand) Synopsis() string {
	return "Configure remote state storage"
}
