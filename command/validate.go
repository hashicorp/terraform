package command

import (
	"flag"
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

func (c *ValidateCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
	var dirPath string

	cmdFlags := flag.NewFlagSet("validate", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

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

	rtnCode := c.validate(dir)

	return rtnCode
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [path]

  Reads the Terraform files in the given path (directory) and
  validates their syntax and basic semantics.

  This is not a full validation that is normally done with
  a plan or apply operation, but can be used to verify the basic
  syntax and usage of Terraform configurations is correct.

Options:

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *ValidateCommand) validate(dir string) int {
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
	return 0
}
