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

type PushCommand struct {
	Meta
}

func (c *PushCommand) Run(args []string) int {
	var force bool
	var statePath, backupPath string
	var remoteConf terraform.RemoteState
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("push", flag.ContinueOnError)
	cmdFlags.StringVar(&statePath, "state", "", "path")
	cmdFlags.StringVar(&backupPath, "backup", "", "path")
	cmdFlags.StringVar(&remoteConf.Name, "remote", "", "")
	cmdFlags.StringVar(&remoteConf.Server, "remote-server", "", "")
	cmdFlags.StringVar(&remoteConf.AuthToken, "remote-auth", "", "")
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

	// Check for the default state file if not specified
	if statePath == "" {
		statePath = DefaultStateFilename
	}

	// Check if an alternative state file exists
	raw, err := ioutil.ReadFile(statePath)
	if err != nil {
		// Ignore if the state path does not exist if it is the default
		// state file path, since that means the user didn't provide any
		// input.
		if !(os.IsNotExist(err) && statePath == DefaultStateFilename) {
			c.Ui.Error(fmt.Sprintf("Failed to open state file at '%s': %v",
				statePath, err))
			return 1
		}
	}

	// Check if both state files are provided!
	if local != nil && raw != nil {
		c.Ui.Error(fmt.Sprintf(`Remote state enabled and default state file is also present.
Please rename the state file at '%s' to prevent a conflict.`, statePath))
		return 1
	}

	// Check if there is no state to push!
	if local == nil && raw == nil {
		c.Ui.Error("No state to push")
		return 1
	}

	// Handle the initial enabling of remote state
	if local == nil && raw != nil {
		if err := c.enableRemote(&remoteConf, raw, statePath, backupPath); err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}
	}

	return c.doPush(force)
}

// enableRemote is used when we get a state file that is not remote enabled,
// and need to move it into the hidden directory and enable remote storage.
func (c *PushCommand) enableRemote(conf *terraform.RemoteState, rawState []byte,
	statePath, backupPath string) error {
	// If there is no local file, ensure we have the remote
	// state is properly configured
	if conf.Empty() {
		return fmt.Errorf("Missing remote configuration")
	}
	if err := remote.ValidateConfig(conf); err != nil {
		return err
	}

	// Decode the state
	state, err := terraform.ReadState(bytes.NewReader(rawState))
	if err != nil {
		return fmt.Errorf("Failed to decode state file at '%s': %v",
			statePath, err)
	}

	// Backup the state file before we remove it
	if backupPath != "-" {
		// If we don't specify a backup path, default to state out with
		// the extension
		if backupPath == "" {
			backupPath = statePath + DefaultBackupExtention
		}

		log.Printf("[INFO] Writing backup state to: %s", backupPath)
		f, err := os.Create(backupPath)
		if err == nil {
			err = terraform.WriteState(state, f)
			f.Close()
		}
		if err != nil {
			return fmt.Errorf("Error writing backup state file: %s", err)
		}
	}

	// Get the target path for the remote state file
	path, err := remote.HiddenStatePath()
	if err != nil {
		return nil
	}

	// Install the state file in the hidden directory
	state.Remote = conf
	f, err := os.Create(path)
	if err == nil {
		err = terraform.WriteState(state, f)
		f.Close()
	}
	if err != nil {
		return fmt.Errorf("Error copying state file: %s", err)
	}

	// Remove the old state file
	if err := os.Remove(statePath); err != nil {
		return fmt.Errorf("Error removing state file: %s", err)
	}
	return nil
}

// doPush is used to attempt the state push
func (c *PushCommand) doPush(force bool) int {
	// Recover the local state if any
	local, _, err := remote.ReadLocalState()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// Attempt to push the state
	change, err := remote.PushState(local.Remote, force)
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

  Uploads the latest state to the remote server. This command can
  also be used to push an existing state file into a remote server and
  to enable automatic state management.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state" path with
                         ".backup" extension. Set to "-" to disable backup.

  -force                 Forces the upload of the local state, ignoring any
                         conflicts. This should be used carefully, as force pushing
						 can cause remote state information to be lost.

  -remote=name           Name of the state file in the state storage server.
                         Optional, default does not use remote storage.

  -remote-auth=token     Authentication token for state storage server.
                         Optional, defaults to blank.

  -remote-server=url     URL of the remote storage server.

  -state=path            Path to read state. Defaults to "terraform.tfstate"
                         unless remote state is enabled.

`
	return strings.TrimSpace(helpText)
}

func (c *PushCommand) Synopsis() string {
	return "Uploads the the local state to the remote server"
}
