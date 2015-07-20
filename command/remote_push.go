package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state"
)

type RemotePushCommand struct {
	Meta
}

func (c *RemotePushCommand) Run(args []string) int {
	var force bool
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("push", flag.ContinueOnError)
	cmdFlags.BoolVar(&force, "force", false, "")
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

	// Refresh the cache state
	if err := cache.Cache.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Failed to refresh from remote state: %s", err))
		return 1
	}

	// Write it to the real storage
	remote := cache.Durable
	if err := remote.WriteState(cache.Cache.State()); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state: %s", err))
		return 1
	}
	if err := remote.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error saving state: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(
		"[reset][bold][green]State successfully pushed!"))
	return 0
}

func (c *RemotePushCommand) Help() string {
	helpText := `
Usage: terraform push [options]

  Uploads the latest state to the remote server.

Options:

  -no-color              If specified, output won't contain any color.

  -force                 Forces the upload of the local state, ignoring any
                         conflicts. This should be used carefully, as force pushing
						 can cause remote state information to be lost.

`
	return strings.TrimSpace(helpText)
}

func (c *RemotePushCommand) Synopsis() string {
	return "Uploads the local state to the remote server"
}
