// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/local"
	backendPluggable "github.com/hashicorp/terraform/internal/backend/pluggable"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/posener/complete"
)

type WorkspaceNewCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceNewCommand) Run(args []string) int {
	args = c.Meta.process(args)
	envCommandShowWarning(c.Ui, c.LegacyName)

	parsedArgs, diags := arguments.ParseWorkspaceNew(args)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	workspace := parsedArgs.Name

	if !validWorkspaceName(workspace) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, workspace))
		return 1
	}

	// You can't ask to create a workspace when you're overriding the
	// workspace name to be something different.
	if current, isOverridden := c.WorkspaceOverridden(); current != workspace && isOverridden {
		c.Ui.Error(envIsOverriddenNewError)
		return 1
	}

	configPath, err := ModulePath(parsedArgs.Args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the backend
	view := arguments.ViewHuman
	b, bDiags := c.backend(configPath, view)
	diags = diags.Append(bDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		c.Ui.Error(fmt.Sprintf("Failed to get configured named states: %s", wDiags.Err()))
		return 1
	}
	c.showDiagnostics(diags) // output warnings, if any

	for _, ws := range workspaces {
		if workspace == ws {
			c.Ui.Error(fmt.Sprintf(envExists, workspace))
			return 1
		}
	}

	// Create the new workspace
	//
	// In local, remote and remote-state backends obtaining a state manager
	// creates an empty state file for the new workspace as a side-effect.
	//
	// The cloud backend also has logic in StateMgr for creating projects and
	// workspaces if they don't already exist.
	sMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		c.Ui.Error(sDiags.Err().Error())
		return 1
	}

	if l, ok := b.(*local.Local); ok {
		if _, ok := l.Backend.(*backendPluggable.Pluggable); ok {
			// Obtaining the state manager would have not created the state file as a side effect
			// if a pluggable state store is in use.
			//
			// Instead, explicitly create the new workspace by saving an empty state file.
			// We only do this when the backend in use is pluggable, to avoid impacting users
			// of remote-state backends.
			if err := sMgr.WriteState(states.NewState()); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
			if err := sMgr.PersistState(nil); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
		}
	}

	// now set the current workspace locally
	if err := c.SetWorkspace(workspace); err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting new workspace: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		strings.TrimSpace(envCreated), workspace)))

	if parsedArgs.StatePath == "" {
		// if we're not loading a state, then we're done
		return 0
	}

	// load the new Backend state
	stateMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		c.Ui.Error(sDiags.Err().Error())
		return 1
	}

	if parsedArgs.StateLock {
		stateLocker := clistate.NewLocker(parsedArgs.StateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if diags := stateLocker.Lock(stateMgr, "workspace-new"); diags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				c.showDiagnostics(diags)
			}
		}()
	}

	// read the existing state file
	f, err := os.Open(parsedArgs.StatePath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	stateFile, err := statefile.Read(f)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// save the existing state in the new Backend.
	err = stateMgr.WriteState(stateFile.State)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	err = stateMgr.PersistState(nil)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *WorkspaceNewCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		complete.PredictAnything,
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceNewCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-state": complete.PredictFiles("*.tfstate"),
	}
}

func (c *WorkspaceNewCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace new [OPTIONS] NAME

  Create a new Terraform workspace.

Options:

    -lock=false         Don't hold a state lock during the operation. This is
                        dangerous if others might concurrently run commands
                        against the same workspace.

    -lock-timeout=0s    Duration to retry a state lock.

    -state=path         Copy an existing state file into the new workspace.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceNewCommand) Synopsis() string {
	return "Create a new workspace"
}
