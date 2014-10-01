package command

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	Meta
}

func (c *RefreshCommand) Run(args []string) int {
	var statePath, stateOutPath, backupPath string

	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("refresh")
	cmdFlags.StringVar(&statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&backupPath, "backup", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The apply command expacts at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		var err error
		configPath, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	// If we don't specify an output path, default to out normal state
	// path.
	if stateOutPath == "" {
		stateOutPath = statePath
	}

	// If we don't specify a backup path, default to state out with
	// the extension
	if backupPath == "" {
		backupPath = stateOutPath + DefaultBackupExtention
	}

	// Verify that the state path exists. The "ContextArg" function below
	// will actually do this, but we want to provide a richer error message
	// if possible.
	if _, err := os.Stat(statePath); err != nil {
		if os.IsNotExist(err) {
			c.Ui.Error(fmt.Sprintf(
				"The Terraform state file for your infrastructure does not\n"+
					"exist. The 'refresh' command only works and only makes sense\n"+
					"when there is existing state that Terraform is managing. Please\n"+
					"double-check the value given below and try again. If you\n"+
					"haven't created infrastructure with Terraform yet, use the\n"+
					"'terraform apply' command.\n\n"+
					"Path: %s",
				statePath))
			return 1
		}

		c.Ui.Error(fmt.Sprintf(
			"There was an error reading the Terraform state that is needed\n"+
				"for refreshing. The path and error are shown below.\n\n"+
				"Path: %s\n\nError: %s",
			statePath,
			err))
		return 1
	}

	// Build the context based on the arguments given
	ctx, _, err := c.Context(contextOpts{
		Path:      configPath,
		StatePath: statePath,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if c.InputEnabled() {
		if err := ctx.Input(); err != nil {
			c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
			return 1
		}
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	// Create a backup of the state before updating
	if backupPath != "-" && c.state != nil {
		log.Printf("[INFO] Writing backup state to: %s", backupPath)
		f, err := os.Create(backupPath)
		if err == nil {
			err = terraform.WriteState(c.state, f)
			f.Close()
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing backup state file: %s", err))
			return 1
		}
	}

	state, err := ctx.Refresh()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
		return 1
	}

	log.Printf("[INFO] Writing state output to: %s", stateOutPath)
	f, err := os.Create(stateOutPath)
	if err == nil {
		defer f.Close()
		err = terraform.WriteState(state, f)
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	return 0
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options] [dir]

  Update the state file of your infrastructure with metadata that matches
  the physical resources they are tracking.

  This will not modify your infrastructure, but it can modify your
  state file to update metadata. This metadata might cause new changes
  to occur when you generate a plan or call apply next.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -input=true         Ask for input for variables if not directly set.

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.

`
	return strings.TrimSpace(helpText)
}

func (c *RefreshCommand) Synopsis() string {
	return "Update local state file against real resources"
}
