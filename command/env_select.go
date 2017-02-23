package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/mitchellh/cli"
)

type EnvSelectCommand struct {
	Meta
}

func (c *EnvSelectCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("env select")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) == 0 {
		c.Ui.Error("expected NAME.\n")
		return cli.RunResultHelp
	}

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{ConfigPath: configPath})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	name := args[0]

	multi, ok := b.(backend.MultiState)
	if !ok {
		c.Ui.Error(envNotSupported)
		return 1
	}

	states, current, err := multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if current == name {
		return 0
	}

	found := false
	for _, s := range states {
		if name == s {
			found = true
			break
		}
	}

	if !found {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, name))
		return 1
	}

	err = multi.ChangeState(name)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envChanged, name),
		),
	)

	return 0
}

func (c *EnvSelectCommand) Help() string {
	helpText := `
Usage: terraform env select NAME [DIR]

  Change Terraform environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvSelectCommand) Synopsis() string {
	return "Change environments"
}
