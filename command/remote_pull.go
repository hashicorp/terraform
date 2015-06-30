package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state"
)

type RemotePullCommand struct {
	Meta
}

func (c *RemotePullCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("pull", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Read out our state
	s, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read state: %s", err))
		return 1
	}
	localState := s.State()

	// If remote state isn't enabled, it is a problem.
	if !localState.IsRemote() {
		c.Ui.Error("Remote state not enabled!")
		return 1
	}

	// We need the CacheState structure in order to do anything
	var cache *state.CacheState
	if bs, ok := s.(*state.BackupState); ok {
		if cs, ok := bs.Real.(*state.CacheState); ok {
			cache = cs
		}
	}
	if cache == nil {
		c.Ui.Error(fmt.Sprintf(
			"Failed to extract internal CacheState from remote state.\n" +
				"This is an internal error, please report it as a bug."))
		return 1
	}

	// Refresh the state
	if err := cache.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Failed to refresh from remote state: %s", err))
		return 1
	}

	// Use an error exit code if the update was not a success
	change := cache.RefreshResult()
	if !change.SuccessfulPull() {
		c.Ui.Error(fmt.Sprintf("%s", change))
		return 1
	} else {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset][bold][green]%s", change)))
	}

	return 0
}

func (c *RemotePullCommand) Help() string {
	helpText := `
Usage: terraform pull [options]

  Refreshes the cached state file from the remote server.

Options:

  -no-color           If specified, output won't contain any color.
`
	return strings.TrimSpace(helpText)
}

func (c *RemotePullCommand) Synopsis() string {
	return "Refreshes the local state copy from the remote server"
}
