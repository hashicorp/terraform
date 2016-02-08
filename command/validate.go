package command

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform/config"
)

// ValidateCommand is a Command implementation that validates the terraform files
type ValidateCommand struct {
	Meta
}

const defaultPath = "."

func (c *ValidateCommand) Help() string {
	return ""
}

func (c *ValidateCommand) Run(args []string) int {
	args = c.Meta.process(args, false)
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

	rtnCode := c.validate(dir)

	return rtnCode
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
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
