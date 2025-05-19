// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"net/url"
	"strings"

	"github.com/hashicorp/cli"
)

// WorkspaceCommand is a Command Implementation that manipulates workspaces,
// which allow multiple distinct states and variables from a single config.
type WorkspaceCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceCommand) Run(args []string) int {
	c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	cmdFlags := c.Meta.extendedFlagSet("workspace")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	return cli.RunResultHelp
}

func (c *WorkspaceCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace

  new, list, show, select and delete Terraform workspaces.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceCommand) Synopsis() string {
	return "Workspace management"
}

// validWorkspaceName returns true is this name is valid to use as a workspace name.
// Since most named states are accessed via a filesystem path or URL, check if
// escaping the name would be required.
func validWorkspaceName(name string) bool {
	return name == url.PathEscape(name)
}

func envCommandShowWarning(ui cli.Ui, show bool) {
	if !show {
		return
	}

	ui.Warn(`Warning: the "terraform env" family of commands is deprecated.

"Workspace" is now the preferred term for what earlier Terraform versions
called "environment", to reduce ambiguity caused by the latter term colliding
with other concepts.

The "terraform workspace" commands should be used instead. "terraform env"
will be removed in a future Terraform version.
`)
}

const (
	envExists = `Workspace %q already exists`

	envDoesNotExist = `
Workspace %q doesn't exist.

You can create this workspace with the "new" subcommand 
or include the "-or-create" flag with the "select" subcommand.`

	envChanged = `[reset][green]Switched to workspace %q.`

	envCreated = `
[reset][green][bold]Created and switched to workspace %q![reset][green]

You're now on a new, empty workspace. Workspaces isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
`

	envDeleted = `[reset][green]Deleted workspace %q!`

	envWarnNotEmpty = `[reset][yellow]WARNING: %q was non-empty.
The resources managed by the deleted workspace may still exist,
but are no longer manageable by Terraform since the state has
been deleted.
`

	envDelCurrent = `
Workspace %[1]q is your active workspace.

You cannot delete the currently active workspace. Please switch
to another workspace and try again.
`

	envInvalidName = `
The workspace name %q is not allowed. The name must contain only URL safe
characters, and no path separators.
`

	envIsOverriddenNote = `

The active workspace is being overridden using the TF_WORKSPACE environment
variable.
`

	envIsOverriddenSelectError = `
The selected workspace is currently overridden using the TF_WORKSPACE
environment variable.

To select a new workspace, either update this environment variable or unset
it and then run this command again.
`

	envIsOverriddenNewError = `
The workspace is currently overridden using the TF_WORKSPACE environment
variable. You cannot create a new workspace when using this setting.

To create a new workspace, either unset this environment variable or update it
to match the workspace name you are trying to create, and then run this command
again.
`
)
