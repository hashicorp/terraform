package command

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	Meta
}

func (c *RefreshCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("refresh")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", 0, "parallelism")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The refresh command expects at most one argument.")
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

	// Check if remote state is enabled
	state, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Verify that the state path exists. The "ContextArg" function below
	// will actually do this, but we want to provide a richer error message
	// if possible.
	if !state.State().IsRemote() {
		if _, err := os.Stat(c.Meta.statePath); err != nil {
			if os.IsNotExist(err) {
				c.Ui.Error(fmt.Sprintf(
					"The Terraform state file for your infrastructure does not\n"+
						"exist. The 'refresh' command only works and only makes sense\n"+
						"when there is existing state that Terraform is managing. Please\n"+
						"double-check the value given below and try again. If you\n"+
						"haven't created infrastructure with Terraform yet, use the\n"+
						"'terraform apply' command.\n\n"+
						"Path: %s",
					c.Meta.statePath))
				return 1
			}

			c.Ui.Error(fmt.Sprintf(
				"There was an error reading the Terraform state that is needed\n"+
					"for refreshing. The path and error are shown below.\n\n"+
					"Path: %s\n\nError: %s",
				c.Meta.statePath,
				err))
			return 1
		}
	}

	// Build the context based on the arguments given
	ctx, _, err := c.Context(contextOpts{
		Path:        configPath,
		StatePath:   c.Meta.statePath,
		Parallelism: c.Meta.parallelism,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := ctx.Input(c.InputMode()); err != nil {
		c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
		return 1
	}

	if !validateContext(ctx, c.Ui) {
		return 1
	}

	newState, err := ctx.Refresh()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
		return 1
	}

	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := c.Meta.PersistState(newState); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	if outputs := outputsAsString(newState, ctx.Module().Config().Outputs); outputs != "" {
		c.Ui.Output(c.Colorize().Color(outputs))
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

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

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
