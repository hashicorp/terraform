// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/backend/local"
	backendPluggable "github.com/hashicorp/terraform/internal/backend/pluggable"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceNewCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceNewCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	// Currently there are no arguments for this command, so ignore the returned value for now.
	args, argDiags := arguments.ParseWorkspaceNew(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewWorkspaceNew(args.ViewType, c.View)

	// Warn against using `terraform env` commands, if needed
	diags = diags.Append(envCommandWarningDiag(c.LegacyName))

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return cli.RunResultHelp
	}

	workspace := args.Name

	// You can't ask to create a workspace when you're overriding the
	// workspace name to be something different.
	//
	// Any errors about the ENV's value we can ignore as we're erroring
	// already due to it being set.
	current, isOverridden, _ := c.WorkspaceOverridden()
	if current != workspace && isOverridden {
		err := errors.New(envIsOverriddenNewError)
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	configPath := c.WorkingDir.RootModuleDir()
	b, bDiags := c.backend(configPath, args.ViewType)
	diags = diags.Append(bDiags)
	if bDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	workspaces, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	for _, ws := range workspaces {
		if workspace == ws {
			err := fmt.Errorf(envExists, workspace)
			diags = diags.Append(err)
			view.Diagnostics(diags)
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
	diags = diags.Append(sDiags)
	if sDiags.HasErrors() {
		view.Diagnostics(diags)
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
				diags = diags.Append(err)
				view.Diagnostics(diags)
				return 1
			}
			if err := sMgr.PersistState(nil); err != nil {
				diags = diags.Append(err)
				view.Diagnostics(diags)
				return 1
			}
		}
	}

	// now set the current workspace locally
	if err := c.SetWorkspace(workspace); err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	view.LogWorkspaceCreationSuccess(workspace, diags)

	if args.StatePath == "" {
		// if we're not loading a state, then we're done
		return 0
	}

	// load the new Backend state
	stateMgr, sDiags := b.StateMgr(workspace)
	diags = diags.Append(sDiags)
	if sDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	if args.Lock {
		stateLocker := clistate.NewLocker(args.LockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if diags := stateLocker.Lock(stateMgr, "workspace-new"); diags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				view.Diagnostics(diags)
			}
		}()
	}

	// read the existing state file
	f, err := os.Open(args.StatePath)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// save the existing state in the new Backend.
	err = stateMgr.WriteState(stateFile.State)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}
	err = stateMgr.PersistState(nil)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
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
