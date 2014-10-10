package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/remote"
	"github.com/hashicorp/terraform/terraform"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	var remoteConf terraform.RemoteState
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("init", flag.ContinueOnError)
	cmdFlags.StringVar(&remoteConf.Name, "remote", "", "")
	cmdFlags.StringVar(&remoteConf.Server, "remote-server", "", "")
	cmdFlags.StringVar(&remoteConf.AuthToken, "remote-auth", "", "")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var path string
	args = cmdFlags.Args()
	if len(args) > 2 {
		c.Ui.Error("The init command expects at most two arguments.\n")
		cmdFlags.Usage()
		return 1
	} else if len(args) < 1 {
		c.Ui.Error("The init command expects at least one arguments.\n")
		cmdFlags.Usage()
		return 1
	}

	if len(args) == 2 {
		path = args[1]
	} else {
		var err error
		path, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	source := args[0]

	// Get our pwd since we need it
	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error reading working directory: %s", err))
		return 1
	}

	// Verify the directory is empty
	if empty, err := config.IsEmptyDir(path); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error checking on destination path: %s", err))
		return 1
	} else if !empty {
		c.Ui.Error(
			"The destination path has Terraform configuration files. The\n" +
				"init command can only be used on a directory without existing Terraform\n" +
				"files.")
		return 1
	}

	// Detect
	source, err = module.Detect(source, pwd)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error with module source: %s", err))
		return 1
	}

	// Get it!
	if err := module.GetCopy(path, source); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Handle remote state if configured
	if !remoteConf.Empty() {
		// Ensure remote state is not already enabled
		haveLocal, err := remote.HaveLocalState()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to check for local state: %v", err))
			return 1
		}
		if haveLocal {
			c.Ui.Error("Remote state is already enabled. Aborting.")
			return 1
		}

		// Check if we have the non-managed state file
		haveNonManaged, err := remote.ExistsFile(DefaultStateFilename)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to check for state file: %v", err))
			return 1
		}
		if haveNonManaged {
			c.Ui.Error(fmt.Sprintf("Existing state file '%s' found. Aborting.",
				DefaultStateFilename))
			return 1
		}

		// Initialize a blank state file with remote enabled
		remoteCmd := &RemoteCommand{
			Meta:       c.Meta,
			remoteConf: remoteConf,
		}
		return remoteCmd.initBlankState()
	}
	return 0
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: terraform init [options] SOURCE [PATH]

  Downloads the module given by SOURCE into the PATH. The PATH defaults
  to the working directory. PATH must be empty of any Terraform files.
  Any conflicting non-Terraform files will be overwritten.

  The module downloaded is a copy. If you're downloading a module from
  Git, it will not preserve the Git history, it will only copy the
  latest files.

Options:

  -remote=name           Name of the state file in the state storage server.
                         Optional, default does not use remote storage.

  -remote-auth=token     Authentication token for state storage server.
                         Optional, defaults to blank.

  -remote-server=url     URL of the remote storage server.

`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initializes Terraform configuration from a module"
}
