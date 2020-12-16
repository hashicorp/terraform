package command

import (
	"net/url"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// WorkspaceCommand is a Command Implementation that manipulates workspaces,
// which allow multiple distinct states and variables from a single config.
type WorkspaceCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceCommand) Run(args []string) int {
	c.Meta.process(args)
	workspaceCommandShowWarning(c.Ui, c.Colorize(), c.LegacyName)

	cmdFlags := c.Meta.extendedFlagSet("workspace")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	c.Ui.Output(c.Help())
	return 0
}

func (c *WorkspaceCommand) Help() string {
	helpText := `
Usage: terraform workspace

  This command has some subcommands that are deprecated aliases for commands
  under "terraform state". Don't use these subcommands for new systems.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceCommand) Synopsis() string {
	return "Deprecated workspace management commands"
}

// validWorkspaceName returns true is this name is valid to use as a workspace name.
// Since most named states are accessed via a filesystem path or URL, check if
// escaping the name would be required.
func validWorkspaceName(name string) bool {
	return name == url.PathEscape(name)
}

func workspaceCommandShowWarning(ui cli.Ui, color *colorstring.Colorize, show bool) {
	if !show {
		return
	}

	ui.Warn(color.Color(`[bold][yellow]Warning:[reset][bold] the "terraform workspace" family of commands is deprecated.[reset]

"State" is now the preferred term for what earlier Terraform versions
called either "workspace" or "environment", to reflect that it's only a
mechanism for having multiple states associated with your configuration and
to reduce confusion with Terraform Cloud's "workspace" concept.

Use subcommands under "terraform satte" instead. "terraform workspace" will
be removed in a future Terraform version.
`))
}

const (
	envExists = `There is already a state named %q`

	envDoesNotExist = `
There is no state named %q.

You can create this state with the "new" subcommand.`

	envChanged = `[reset][green]Switched to state %q.`

	envCreated = `
[reset][green][bold]Created and switched to state %q![reset]

You're now using a new, empty state that is separated from the state
you had previously selected. If you run "terraform plan" now, Terraform
will plan to create new objects for all of the resources declared in
your configuration.
`

	envDeleted = `[reset][green]Deleted state %q.`

	envNotEmpty = `
State %[1]q is currently tracking resources.

Deleting this state might result in remote objects that still exist but that
are no longer tracked by Terraform. Destroy these resources first.

If you want to delete this state anyway and risk Terraform forgetting about
these remote objects, use the "-force" option.
`

	envWarnNotEmpty = `[bold][reset][yellow]Warning:[reset][bold] State %q was tracking resources.[reset]

Any remote objects represented by those resources still exist in the remote
system, but Terraform has no record of them. If you wish to delete them then
you must do so outside of Terraform.
`

	envDelCurrent = `
State %[1]q is your current state.

You cannot delete the currently-selected state. Select a different named state
first, before deleting this state.
`

	envInvalidName = `
The name %q is not a valid state name. A state name must contain only URL-safe
characters and no path separators.
`

	envIsOverriddenNote = `

The current state is being overridden using the TF_CURRENT_STATE environment
variable.
`
	envIsOverriddenNoteLegacy = `

The current state is being overridden using the TF_WORKSPACE environment
variable, which is deprecated but still supported. Use TF_CURRENT_STATE to
override the current state instead.
`

	envIsOverriddenSelectError = `
The current state name is overridden using the TF_CURRENT_STATE environment
variable.

To select a new workspace, either update that environment variable or unset
it and then run this command again.
`

	envIsOverriddenSelectErrorLegacy = `
The current state name is overridden using the deprecated TF_WORKSPACE
environment variable.

To select a new workspace, either update that environment variable or unset
it and then run this command again.
`

	envIsOverriddenNewError = `
The current state name is overridden using the TF_CURRENT_STATE environment
variable. You cannot create a new workspace when forcing a state via the
environment.

To create a new workspace, either unset this environment variable or update it
to match the workspace name you are trying to create, and then run this command
again.
`

	envIsOverriddenNewErrorLegacy = `
The current state name is overridden using the deprecated TF_WORKSPACE
environment variable. You cannot create a new workspace when forcing a state
via the environment.

To create a new workspace, either unset this environment variable or update it
to match the workspace name you are trying to create, and then run this command
again.
`
)
