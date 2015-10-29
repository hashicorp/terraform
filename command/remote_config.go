package command

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// remoteCommandConfig is used to encapsulate our configuration
type remoteCommandConfig struct {
	disableRemote bool
	pullOnDisable bool

	statePath  string
	backupPath string
}

// RemoteConfigCommand is a Command implementation that is used to
// enable and disable remote state management
type RemoteConfigCommand struct {
	Meta
	conf       remoteCommandConfig
	remoteConf terraform.RemoteState
}

func (c *RemoteConfigCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
	config := make(map[string]string)
	cmdFlags := flag.NewFlagSet("remote", flag.ContinueOnError)
	cmdFlags.BoolVar(&c.conf.disableRemote, "disable", false, "")
	cmdFlags.BoolVar(&c.conf.pullOnDisable, "pull", true, "")
	cmdFlags.StringVar(&c.conf.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.conf.backupPath, "backup", "", "path")
	cmdFlags.StringVar(&c.remoteConf.Type, "backend", "atlas", "")
	cmdFlags.Var((*FlagKV)(&config), "backend-config", "config")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("\nError parsing CLI flags: %s", err))
		return 1
	}

	// Lowercase the type
	c.remoteConf.Type = strings.ToLower(c.remoteConf.Type)

	// Set the local state path
	c.statePath = c.conf.statePath

	// Populate the various configurations
	c.remoteConf.Config = config

	// Get the state information. We specifically request the cache only
	// for the remote state here because it is possible the remote state
	// is invalid and we don't want to error.
	stateOpts := c.StateOpts()
	stateOpts.RemoteCacheOnly = true
	if _, err := c.StateRaw(stateOpts); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading local state: %s", err))
		return 1
	}

	// Get the local and remote [cached] state
	localState := c.stateResult.Local.State()
	var remoteState *terraform.State
	if remote := c.stateResult.Remote; remote != nil {
		remoteState = remote.State()
	}

	// Check if remote state is being disabled
	if c.conf.disableRemote {
		if !remoteState.IsRemote() {
			c.Ui.Error(fmt.Sprintf("Remote state management not enabled! Aborting."))
			return 1
		}
		if !localState.Empty() {
			c.Ui.Error(fmt.Sprintf("State file already exists at '%s'. Aborting.",
				c.conf.statePath))
			return 1
		}

		return c.disableRemoteState()
	}

	// Ensure there is no conflict, and then do the correct operation
	var result int
	haveCache := !remoteState.Empty()
	haveLocal := !localState.Empty()
	switch {
	case haveCache && haveLocal:
		c.Ui.Error(fmt.Sprintf("Remote state is enabled, but non-managed state file '%s' is also present!",
			c.conf.statePath))
		result = 1

	case !haveCache && !haveLocal:
		// If we don't have either state file, initialize a blank state file
		result = c.initBlankState()

	case haveCache && !haveLocal:
		// Update the remote state target potentially
		result = c.updateRemoteConfig()

	case !haveCache && haveLocal:
		// Enable remote state management
		result = c.enableRemoteState()
	}

	// If there was an error, return right away
	if result != 0 {
		return result
	}

	// If we're not pulling, then do nothing
	if !c.conf.pullOnDisable {
		return result
	}

	// Otherwise, refresh the state
	stateResult, err := c.StateRaw(c.StateOpts())
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error while performing the initial pull. The error message is shown\n"+
				"below. Note that remote state was properly configured, so you don't\n"+
				"need to reconfigure. You can now use `push` and `pull` directly.\n"+
				"\n%s", err))
		return 1
	}

	state := stateResult.State
	if err := state.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error while performing the initial pull. The error message is shown\n"+
				"below. Note that remote state was properly configured, so you don't\n"+
				"need to reconfigure. You can now use `push` and `pull` directly.\n"+
				"\n%s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		"[reset][bold][green]Remote state configured and pulled.")))
	return 0
}

// disableRemoteState is used to disable remote state management,
// and move the state file into place.
func (c *RemoteConfigCommand) disableRemoteState() int {
	if c.stateResult == nil {
		c.Ui.Error(fmt.Sprintf(
			"Internal error. State() must be called internally before remote\n" +
				"state can be disabled. Please report this as a bug."))
		return 1
	}
	if !c.stateResult.State.State().IsRemote() {
		c.Ui.Error(fmt.Sprintf(
			"Remote state is not enabled. Can't disable remote state."))
		return 1
	}
	local := c.stateResult.Local
	remote := c.stateResult.Remote

	// Ensure we have the latest state before disabling
	if c.conf.pullOnDisable {
		log.Printf("[INFO] Refreshing local state from remote server")
		if err := remote.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Failed to refresh from remote state: %s", err))
			return 1
		}

		// Exit if we were unable to update
		if change := remote.RefreshResult(); !change.SuccessfulPull() {
			c.Ui.Error(fmt.Sprintf("%s", change))
			return 1
		} else {
			log.Printf("[INFO] %s", change)
		}
	}

	// Clear the remote management, and copy into place
	newState := remote.State()
	newState.Remote = nil
	if err := local.WriteState(newState); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to encode state file '%s': %s",
			c.conf.statePath, err))
		return 1
	}
	if err := local.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to encode state file '%s': %s",
			c.conf.statePath, err))
		return 1
	}

	// Remove the old state file
	if err := os.Remove(c.stateResult.RemotePath); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to remove the local state file: %v", err))
		return 1
	}

	return 0
}

// validateRemoteConfig is used to verify that the remote configuration
// we have is valid
func (c *RemoteConfigCommand) validateRemoteConfig() error {
	conf := c.remoteConf
	_, err := remote.NewClient(conf.Type, conf.Config)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"%s\n\n"+
				"If the error message above mentions requiring or modifying configuration\n"+
				"options, these are set using the `-backend-config` flag. Example:\n"+
				"-backend-config=\"name=foo\" to set the `name` configuration",
			err))
	}
	return err
}

// initBlank state is used to initialize a blank state that is
// remote enabled
func (c *RemoteConfigCommand) initBlankState() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Make a blank state, attach the remote configuration
	blank := terraform.NewState()
	blank.Remote = &c.remoteConf

	// Persist the state
	remote := &state.LocalState{Path: c.stateResult.RemotePath}
	if err := remote.WriteState(blank); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to initialize state file: %v", err))
		return 1
	}
	if err := remote.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to initialize state file: %v", err))
		return 1
	}

	// Success!
	c.Ui.Output("Initialized blank state with remote state enabled!")
	return 0
}

// updateRemoteConfig is used to update the configuration of the
// remote state store
func (c *RemoteConfigCommand) updateRemoteConfig() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Read in the local state, which is just the cache of the remote state
	remote := c.stateResult.Remote.Cache

	// Update the configuration
	state := remote.State()
	state.Remote = &c.remoteConf
	if err := remote.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	if err := remote.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Success!
	c.Ui.Output("Remote configuration updated")
	return 0
}

// enableRemoteState is used to enable remote state management
// and to move a state file into place
func (c *RemoteConfigCommand) enableRemoteState() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Read the local state
	local := c.stateResult.Local
	if err := local.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read local state: %s", err))
		return 1
	}

	// Backup the state file before we modify it
	backupPath := c.conf.backupPath
	if backupPath != "-" {
		// Provide default backup path if none provided
		if backupPath == "" {
			backupPath = c.conf.statePath + DefaultBackupExtension
		}

		log.Printf("[INFO] Writing backup state to: %s", backupPath)
		backup := &state.LocalState{Path: backupPath}
		if err := backup.WriteState(local.State()); err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing backup state file: %s", err))
			return 1
		}
		if err := backup.PersistState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing backup state file: %s", err))
			return 1
		}
	}

	// Update the local configuration, move into place
	state := local.State()
	state.Remote = &c.remoteConf
	remote := c.stateResult.Remote
	if err := remote.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	if err := remote.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Remove the original, local state file
	log.Printf("[INFO] Removing state file: %s", c.conf.statePath)
	if err := os.Remove(c.conf.statePath); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to remove state file '%s': %v",
			c.conf.statePath, err))
		return 1
	}

	// Success!
	c.Ui.Output("Remote state management enabled")
	return 0
}

func (c *RemoteConfigCommand) Help() string {
	helpText := `
Usage: terraform remote config [options]

  Configures Terraform to use a remote state server. This allows state
  to be pulled down when necessary and then pushed to the server when
  updated. In this mode, the state file does not need to be stored durably
  since the remote server provides the durability.

Options:

  -backend=Atlas         Specifies the type of remote backend. Must be one
                         of Atlas, Consul, Etcd, HTTP, S3, or Swift. Defaults
                         to Atlas.

  -backend-config="k=v"  Specifies configuration for the remote storage
                         backend. This can be specified multiple times.

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state" path with
                         ".backup" extension. Set to "-" to disable backup.

  -disable               Disables remote state management and migrates the state
                         to the -state path.

  -pull=true             If disabling, this controls if the remote state is
                         pulled before disabling. If enabling, this controls
                         if the remote state is pulled after enabling. This
                         defaults to true.

  -state=path            Path to read state. Defaults to "terraform.tfstate"
                         unless remote state is enabled.

  -no-color              If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *RemoteConfigCommand) Synopsis() string {
	return "Configures remote state management"
}
