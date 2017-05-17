package command

import (
	"bytes"
	"fmt"
	"strings"
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

	args = cmdFlags.Args()
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

	states, err := b.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	env := c.Env()

	var out bytes.Buffer
	for _, s := range states {
		if s == env {
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
