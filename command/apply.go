package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	TFConfig *terraform.Config
	Ui       cli.Ui
}

func (c *ApplyCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("apply", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error(
			"The apply command expects only one argument with the path\n" +
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

	tf, err := terraform.New(tfconfig)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing Terraform: %s", err))
		return 1
	}

	diff, err := tf.Diff(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running diff: %s", err))
		return 1
	}

	state, err := tf.Apply(nil, diff)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error applying diff: %s", err))
		return 1
	}

	c.Ui.Output(strings.TrimSpace(state.String()))

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
