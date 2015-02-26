package command

import (
	"fmt"
	"os"
	"strings"
)

// ValidateCommand is a cli.Command implementation that validates the
// Terraform configuration files.
type ValidateCommand struct {
	Meta
}

func (c *ValidateCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("validate")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The validate command expects at most one argument.")
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

	// Build the context based on the arguments given
	ctx, _, err := c.Context(contextOpts{
		Path:      configPath,
		StatePath: c.Meta.statePath,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	c.Ui.Info("Terraform configuration valid.")
	return 0
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [dir]

  Validate all terraform files in the specified directory.

Options:

  -input=true         Ask for input for variables if not directly set.

  -no-color           If specified, output won't contain any color.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.

`
	return strings.TrimSpace(helpText)
}

func (c *ValidateCommand) Synopsis() string {
	return "Validate configuration files"
}
