package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
)

type EnvListCommand struct {
	Meta
}

func (c *EnvListCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("env list")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(args)
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

	var out bytes.Buffer
	for _, s := range states {
		if s == current {
			out.WriteString("* ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(s + "\n")
	}

	c.Ui.Output(out.String())
	return 0
}

func (c *EnvListCommand) Help() string {
	helpText := `
Usage: terraform env list [DIR]

  List Terraform environments.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvListCommand) Synopsis() string {
	return "List Environments"
}
