package command

import (
	"net/url"
	"strings"
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

// validEnvName returns true is this name is valid to use as an environment name.
// Since most named states are accessed via a filesystem path or URL, check if
// escaping the name would be required.
func validEnvName(name string) bool {
	return name == url.PathEscape(name)
}

const (
	envNotSupported = `Backend does not support environments`

	envExists = `Environment %q already exists`

	envDoesNotExist = `
Environment %q doesn't exist!

You can create this environment with the "new" option.`

	envChanged = `[reset][green]Switched to environment %q!`

	envCreated = `
[reset][green][bold]Created and switched to environment %q![reset][green]

You're now on a new, empty environment. Environments isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
`

	envDeleted = `[reset][green]Deleted environment %q!`

	envNotEmpty = `
Environment %[1]q is not empty!

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

	envDelCurrent = `
Environment %[1]q is your active environment!

You cannot delete the currently active environment. Please switch
to another environment and try again.
`

	envInvalidName = `
The environment name %q is not allowed. The name must contain only URL safe
characters, and no path separators.
`
)
