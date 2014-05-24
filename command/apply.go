package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Ui cli.Ui
}

func (c *ApplyCommand) Run(_ []string) int {
	return 0
}

func (c *ApplyCommand) Help() string {
	helpText := `
Usage: terraform apply [terraform.tf]

  Builds or changes infrastructure according to the Terraform configuration
  file.

Options:

  -init   If specified, it is okay to build brand new infrastructure
          (with no state file specified).

`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) Synopsis() string {
	return "Builds or changes infrastructure according to Terrafiles"
}
