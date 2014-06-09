package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// DiffCommand is a Command implementation that compares a Terraform
// configuration to an actual infrastructure and shows the differences.
type DiffCommand struct {
	TFConfig *terraform.Config
	Ui       cli.Ui
}

func (c *DiffCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("diff", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error(
			"The diff command expects only one argument with the path\n" +
				"to a Terraform configuration.\n")
		cmdFlags.Usage()
		return 1
	}

	b, err := config.Load(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading blueprint: %s", err))
		return 1
	}

	tfconfig := c.TFConfig
	tfconfig.Config = b

	_, err = terraform.New(tfconfig)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing Terraform: %s", err))
		return 1
	}

	return 0
}

func (c *DiffCommand) Help() string {
	helpText := `
Usage: terraform diff [options] [terraform.tf]

  Shows the differences between the Terraform configuration and
  the actual state of an infrastructure.

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources.

`
	return strings.TrimSpace(helpText)
}

func (c *DiffCommand) Synopsis() string {
	return "Show changes between Terraform config and infrastructure"
}
