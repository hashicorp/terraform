package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// ValidateCommand is a Command implementation that validates the terraform files
type ValidateCommand struct {
	Meta
}

const defaultPath = "."

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [dir]

  Validate the terraform files in a directory. If dir is not specified,
  then the current directory will be used.

Options:

  -check-vars         If specified, the command will check that all the
                      variables without defaults in the configuration are
                      specified.

  -no-color           If specified, output won't contain any color.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.
`
	return strings.TrimSpace(helpText)
}

func (c *ValidateCommand) Run(args []string) int {
	args = c.Meta.process(args, true)
	var checkVars bool

	cmdFlags := c.Meta.flagSet("validate")
	cmdFlags.BoolVar(&checkVars, "check-vars", false, "check-vars")
	cmdFlags.Usage = func() {
		c.Ui.Error(c.Help())
	}
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()

	var dirPath string
	if len(args) == 1 {
		dirPath = args[0]
	} else {
		dirPath = "."
	}
	dir, err := filepath.Abs(dirPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Unable to locate directory %v\n", err.Error()))
	}

	rtnCode := c.validate(dir, checkVars)

	return rtnCode
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
}

func (c *ValidateCommand) validate(dir string, checkVars bool) int {
	cfg, err := config.LoadDir(dir)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error loading files %v\n", err.Error()))
		return 1
	}
	err = cfg.Validate()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error validating: %v\n", err.Error()))
		return 1
	}

	if checkVars {
		context, _, err := c.Context(contextOpts{
			Path: dir,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %v\n", err.Error()))
			return 1
		}
		if !validateContext(context, c.Ui) {
			return 1
		}
	}

	return 0
}
