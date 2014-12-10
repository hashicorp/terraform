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
	var remoteBackend, remoteAddress, remoteAccessToken, remoteName, remotePath string
	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("init", flag.ContinueOnError)
	cmdFlags.StringVar(&remoteBackend, "backend", "atlas", "")
	cmdFlags.StringVar(&remoteAddress, "address", "", "")
	cmdFlags.StringVar(&remoteAccessToken, "access-token", "", "")
	cmdFlags.StringVar(&remoteName, "name", "", "")
	cmdFlags.StringVar(&remotePath, "path", "", "")
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
	if remoteAddress != "" || remoteAccessToken != "" || remoteName != "" || remotePath != "" {
		var remoteConf terraform.RemoteState
		remoteConf.Type = remoteBackend
		remoteConf.Config = map[string]string{
			"address":      remoteAddress,
			"access_token": remoteAccessToken,
			"name":         remoteName,
			"path":         remotePath,
		}

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

  -address=url           URL of the remote storage server.
                         Required for HTTP backend, optional for Atlas and Consul.

  -access-token=token    Authentication token for state storage server.
                         Required for Atlas backend, optional for Consul.

  -backend=atlas         Specifies the type of remote backend. Must be one
                         of Atlas, Consul, or HTTP. Defaults to atlas.

  -name=name             Name of the state file in the state storage server.
                         Required for Atlas backend.

  -path=path             Path of the remote state in Consul. Required for the
                         Consul backend.

`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initializes Terraform configuration from a module"
}
