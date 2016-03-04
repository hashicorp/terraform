package command

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	var remoteBackend string
	args = c.Meta.process(args, false)
	remoteConfig := make(map[string]string)
	cmdFlags := flag.NewFlagSet("init", flag.ContinueOnError)
	cmdFlags.StringVar(&remoteBackend, "backend", "", "")
	cmdFlags.Var((*FlagKV)(&remoteConfig), "backend-config", "config")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	remoteBackend = strings.ToLower(remoteBackend)

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

	// Set the state out path to be the path requested for the module
	// to be copied. This ensures any remote states gets setup in the
	// proper directory.
	c.Meta.dataDir = filepath.Join(path, DefaultDataDirectory)

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
	source, err = getter.Detect(source, pwd, getter.Detectors)
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
	if remoteBackend != "" {
		var remoteConf terraform.RemoteState
		remoteConf.Type = remoteBackend
		remoteConf.Config = remoteConfig

		state, err := c.State()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error checking for state: %s", err))
			return 1
		}
		if state != nil {
			s := state.State()
			if !s.Empty() {
				c.Ui.Error(fmt.Sprintf(
					"State file already exists and is not empty! Please remove this\n" +
						"state file before initializing. Note that removing the state file\n" +
						"may result in a loss of information since Terraform uses this\n" +
						"to track your infrastructure."))
				return 1
			}
			if s.IsRemote() {
				c.Ui.Error(fmt.Sprintf(
					"State file already exists with remote state enabled! Please remove this\n" +
						"state file before initializing. Note that removing the state file\n" +
						"may result in a loss of information since Terraform uses this\n" +
						"to track your infrastructure."))
				return 1
			}
		}

		// Initialize a blank state file with remote enabled
		remoteCmd := &RemoteConfigCommand{
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

  -backend=atlas         Specifies the type of remote backend. If not
                         specified, local storage will be used.

  -backend-config="k=v"  Specifies configuration for the remote storage
                         backend. This can be specified multiple times.

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initializes Terraform configuration from a module"
}
