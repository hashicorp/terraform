// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
//
// This has been moved to the arguments package but also kept here, as the function is used
// in some places for reasons other than validating command-line arguments.
// TODO: Decide whether to keep this function or use arguments package to validate ENV values too.
func validWorkspaceName(name string) bool {
	return arguments.ValidWorkspaceName(name)
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

// envCommandWarningDiag returns diagnostic version of the output from envCommandShowWarning.
// This should be used when the output is being rendered in a machine-readable format.
func envCommandWarningDiag(show bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if !show {
		return diags // empty
	}

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Detail:   `Warning: the "terraform env" family of commands is deprecated.`,
		Summary: `"Workspace" is now the preferred term for what earlier Terraform versions
called "environment", to reduce ambiguity caused by the latter term colliding
with other concepts.

The "terraform workspace" commands should be used instead. "terraform env"
will be removed in a future Terraform version.
`,
	})
	return diags
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

// warnNoEnvsExistDiag creates a warning diagnostic saying that no workspaces exist,
// and provides guidance about how to create the workspace based on whether the workspace is
// custom or not.
func warnNoEnvsExistDiag(currentWorkspace string) tfdiags.Diagnostic {
	summary := "Terraform cannot find any existing workspaces."

	if currentWorkspace == backend.DefaultStateName {
		// Recommended actions for the user includes running `init` if they're using the default workspace.
		msg := fmt.Sprintf(
			"The %q workspace is selected in your working directory. You can create this workspace by running \"terraform init\", by using the \"terraform workspace new\" subcommand or by including the \"-or-create\" flag with the \"terraform workspace select\" subcommand.",
			currentWorkspace,
		)
		return tfdiags.Sourceless(
			tfdiags.Warning,
			summary,
			msg,
		)
	}

	msg := fmt.Sprintf(
		"The %q workspace is selected in your working directory. You can create this workspace by using the \"terraform workspace new\" subcommand or including the \"-or-create\" flag with the \"terraform workspace select\" subcommand.",
		currentWorkspace,
	)
	return tfdiags.Sourceless(
		tfdiags.Warning,
		summary,
		msg,
	)
}
