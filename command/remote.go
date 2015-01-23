package command

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
)

// remoteCommandConfig is used to encapsulate our configuration
type remoteCommandConfig struct {
	disableRemote bool
	pullOnDisable bool

	statePath  string
	backupPath string
}

// RemoteCommand is a Command implementation that is used to
// enable and disable remote state management
type RemoteCommand struct {
	Meta
	conf       remoteCommandConfig
	remoteConf terraform.RemoteState
}

func (c *RemoteCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
	var address, accessToken, name, path string
	cmdFlags := flag.NewFlagSet("remote", flag.ContinueOnError)
	cmdFlags.BoolVar(&c.conf.disableRemote, "disable", false, "")
	cmdFlags.BoolVar(&c.conf.pullOnDisable, "pull", true, "")
	cmdFlags.StringVar(&c.conf.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.conf.backupPath, "backup", "", "path")
	cmdFlags.StringVar(&c.remoteConf.Type, "backend", "atlas", "")
	cmdFlags.StringVar(&address, "address", "", "")
	cmdFlags.StringVar(&accessToken, "access-token", "", "")
	cmdFlags.StringVar(&name, "name", "", "")
	cmdFlags.StringVar(&path, "path", "", "")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Show help if given no inputs
	if !c.conf.disableRemote && c.remoteConf.Type == "atlas" &&
		name == "" && accessToken == "" {
		cmdFlags.Usage()
		return 1
	}

	// Populate the various configurations
	c.remoteConf.Config = map[string]string{
		"address":      address,
		"access_token": accessToken,
		"name":         name,
		"path":         path,
	}

	// Check if have an existing local state file
	haveLocal, err := remote.HaveLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for local state: %v", err))
		return 1
	}

	// Check if we have the non-managed state file
	haveNonManaged, err := remote.ExistsFile(c.conf.statePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for state file: %v", err))
		return 1
	}

	// Check if remote state is being disabled
	if c.conf.disableRemote {
		if !haveLocal {
			c.Ui.Error(fmt.Sprintf("Remote state management not enabled! Aborting."))
			return 1
		}
		if haveNonManaged {
			c.Ui.Error(fmt.Sprintf("State file already exists at '%s'. Aborting.",
				c.conf.statePath))
			return 1
		}
		return c.disableRemoteState()
	}

	// Ensure there is no conflict
	switch {
	case haveLocal && haveNonManaged:
		c.Ui.Error(fmt.Sprintf("Remote state is enabled, but non-managed state file '%s' is also present!",
			c.conf.statePath))
		return 1

	case !haveLocal && !haveNonManaged:
		// If we don't have either state file, initialize a blank state file
		return c.initBlankState()

	case haveLocal && !haveNonManaged:
		// Update the remote state target potentially
		return c.updateRemoteConfig()

	case !haveLocal && haveNonManaged:
		// Enable remote state management
		return c.enableRemoteState()

	default:
		panic("unhandled case")
	}
	return 0
}

// disableRemoteState is used to disable remote state management,
// and move the state file into place.
func (c *RemoteCommand) disableRemoteState() int {
	// Get the local state
	local, _, err := remote.ReadLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read local state: %v", err))
		return 1
	}

	// Ensure we have the latest state before disabling
	if c.conf.pullOnDisable {
		log.Printf("[INFO] Refreshing local state from remote server")
		change, err := remote.RefreshState(local.Remote)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Failed to refresh from remote state: %v", err))
			return 1
		}

		// Exit if we were unable to update
		if !change.SuccessfulPull() {
			c.Ui.Error(fmt.Sprintf("%s", change))
			return 1
		} else {
			log.Printf("[INFO] %s", change)
		}

		// Reload the local state after the refresh
		local, _, err = remote.ReadLocalState()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to read local state: %v", err))
			return 1
		}
	}

	// Clear the remote management, and copy into place
	local.Remote = nil
	fh, err := os.Create(c.conf.statePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to create state file '%s': %v",
			c.conf.statePath, err))
		return 1
	}
	defer fh.Close()
	if err := terraform.WriteState(local, fh); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to encode state file '%s': %v",
			c.conf.statePath, err))
		return 1
	}

	// Remove the old state file
	path, err := remote.HiddenStatePath()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to get local state path: %v", err))
		return 1
	}
	if err := os.Remove(path); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to remove the local state file: %v", err))
		return 1
	}
	return 0
}

// validateRemoteConfig is used to verify that the remote configuration
// we have is valid
func (c *RemoteCommand) validateRemoteConfig() error {
	err := remote.ValidConfig(&c.remoteConf)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
	}
	return err
}

// initBlank state is used to initialize a blank state that is
// remote enabled
func (c *RemoteCommand) initBlankState() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Make the hidden directory
	if err := remote.EnsureDirectory(); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Make a blank state, attach the remote configuration
	blank := terraform.NewState()
	blank.Remote = &c.remoteConf

	// Persist the state
	if err := remote.PersistState(blank); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to initialize state file: %v", err))
		return 1
	}

	// Success!
	c.Ui.Output("Initialized blank state with remote state enabled!")
	return 0
}

// updateRemoteConfig is used to update the configuration of the
// remote state store
func (c *RemoteCommand) updateRemoteConfig() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Read in the local state
	local, _, err := remote.ReadLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read local state: %v", err))
		return 1
	}

	// Update the configuration
	local.Remote = &c.remoteConf
	if err := remote.PersistState(local); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Success!
	c.Ui.Output("Remote configuration updated")
	return 0
}

// enableRemoteState is used to enable remote state management
// and to move a state file into place
func (c *RemoteCommand) enableRemoteState() int {
	// Validate the remote configuration
	if err := c.validateRemoteConfig(); err != nil {
		return 1
	}

	// Make the hidden directory
	if err := remote.EnsureDirectory(); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Read the provided state file
	raw, err := ioutil.ReadFile(c.conf.statePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read '%s': %v", c.conf.statePath, err))
		return 1
	}
	state, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to decode '%s': %v", c.conf.statePath, err))
		return 1
	}

	// Backup the state file before we modify it
	backupPath := c.conf.backupPath
	if backupPath != "-" {
		// Provide default backup path if none provided
		if backupPath == "" {
			backupPath = c.conf.statePath + DefaultBackupExtention
		}

		log.Printf("[INFO] Writing backup state to: %s", backupPath)
		f, err := os.Create(backupPath)
		if err == nil {
			err = terraform.WriteState(state, f)
			f.Close()
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing backup state file: %s", err))
			return 1
		}
	}

	// Update the local configuration, move into place
	state.Remote = &c.remoteConf
	if err := remote.PersistState(state); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Remove the state file
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

func (c *RemoteCommand) Help() string {
	helpText := `
Usage: terraform remote [options]

  Configures Terraform to use a remote state server. This allows state
  to be pulled down when necessary and then pushed to the server when
  updated. In this mode, the state file does not need to be stored durably
  since the remote server provides the durability.

Options:

  -address=url           URL of the remote storage server.
                         Required for HTTP backend, optional for Atlas and Consul.

  -access-token=token    Authentication token for state storage server.
                         Required for Atlas backend, optional for Consul.

  -backend=Atlas         Specifies the type of remote backend. Must be one
                         of Atlas, Consul, or HTTP. Defaults to Atlas.

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state" path with
                         ".backup" extension. Set to "-" to disable backup.

  -disable               Disables remote state management and migrates the state
                         to the -state path.

  -name=name             Name of the state file in the state storage server.
                         Required for Atlas backend.

  -path=path             Path of the remote state in Consul. Required for the
                         Consul backend.

  -pull=true             Controls if the remote state is pulled before disabling.
                         This defaults to true to ensure the latest state is cached
						 before disabling.

  -state=path            Path to read state. Defaults to "terraform.tfstate"
                         unless remote state is enabled.

`
	return strings.TrimSpace(helpText)
}

func (c *RemoteCommand) Synopsis() string {
	return "Configures remote state management"
}
