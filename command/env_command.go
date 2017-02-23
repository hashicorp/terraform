package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/mitchellh/cli"
)

// EnvCommand is a Command Implementation that manipulates local state
// environments.
type EnvCommand struct {
	Meta
}

func (c *EnvCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("env")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) > 0 {
		c.Ui.Error("0 arguments expected.\n")
		return cli.RunResultHelp
	}

	// Load the backend
	b, err := c.Backend(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	multi, ok := b.(backend.MultiState)
	if !ok {
		c.Ui.Error(envNotSupported)
		return 1
	}
	_, current, err := multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Current environment is %q\n", current))
	c.Ui.Output(c.Help())
	return 0
}

func (c *EnvCommand) Help() string {
	helpText := `
Usage: terraform env

  Create, change and delete Terraform environments. 


Subcommands:

    list      List environments.
    select    Select an environment.
    new       Create a new environment.
    delete    Delete an existing environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvCommand) Synopsis() string {
	return "Environment management"
}

const (
	envNotSupported = `Backend does not support environments`

	envExists = `Environment %q already exists`

	envDoesNotExist = `Environment %q doesn't exist!
You can create this environment with the "-new" option.`

	envChanged = `[reset][green]Switched to environment %q!`

	envCreated = `[reset][green]Created environment %q!`

	envDeleted = `[reset][green]Deleted environment %q!`

	envNotEmpty = `Environment %[1]q is not empty!
Deleting %[1]q can result in dangling resources: resources that 
exist but are no longer manageable by Terraform. Please destroy
these resources first.  If you want to delete this environment
anyways and risk dangling resources, use the '-force' flag.
`

	envWarnNotEmpty = `[reset][yellow]WARNING: %q was non-empty. 
The resources managed by the deleted environment may still exist,
but are no longer manageable by Terraform since the state has
been deleted.
`
)
