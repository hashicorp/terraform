// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

// StatePushCommand is a Command implementation that allows
// pushing a local state to a remote location.
type StatePushCommand struct {
	Meta
	StateMeta
}

func (c *StatePushCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)

	// Propagate -no-color for legacy Ui usage
	if common.NoColor {
		c.Meta.Color = false
	}
	c.Meta.color = c.Meta.Color

	// Configure the view with the reconciled color setting
	c.View.Configure(&arguments.View{
		NoColor:         !c.Meta.color,
		CompactWarnings: common.CompactWarnings,
	})

	// Parse command flags/args
	args, diags := arguments.ParseStatePush(rawArgs)

	// Instantiate the view even if flag parsing produced errors
	view := views.NewStatePush(c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Copy parsed args to appropriate Meta fields
	c.Meta.stateLock = args.StateLock
	c.Meta.stateLockTimeout = args.StateLockTimeout
	c.Meta.ignoreRemoteVersion = args.IgnoreRemoteVersion

	if diags := c.Meta.checkRequiredVersion(); diags != nil {
		view.Diagnostics(diags)
		return 1
	}

	// Determine our reader for the input state. This is the filepath
	// or stdin if "-" is given.
	var r io.Reader = os.Stdin
	if args.Path != "-" {
		f, err := os.Open(args.Path)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		// Note: we don't need to defer a Close here because we do a close
		// automatically below directly after the read.

		r = f
	}

	// Read the state
	srcStateFile, err := statefile.Read(r)
	if c, ok := r.(io.Closer); ok {
		// Close the reader if possible right now since we're done with it.
		c.Close()
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading source state %q: %s", args.Path, err))
		return 1
	}

	// Load the backend
	viewType := arguments.ViewHuman
	b, backendDiags := c.backend(".", viewType)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Determine the workspace name
	workspace, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, workspace)
	view.Diagnostics(remoteVersionDiags)
	if remoteVersionDiags.HasErrors() {
		return 1
	}

	// Get the state manager for the currently-selected workspace
	stateMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		c.Ui.Error(fmt.Sprintf("Failed to load destination state: %s", sDiags.Err()))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if diags := stateLocker.Lock(stateMgr, "state-push"); diags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				view.Diagnostics(diags)
			}
		}()
	}

	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh destination state: %s", err))
		return 1
	}

	if srcStateFile == nil {
		// We'll push a new empty state instead
		srcStateFile = statemgr.NewStateFile()
	}

	// Import it, forcing through the lineage/serial if requested and possible.
	if err := statemgr.Import(srcStateFile, stateMgr, args.Force); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to write state: %s", err))
		return 1
	}

	// Get schemas, if possible, before writing state
	var schemas *terraform.Schemas
	if isCloudMode(b) {
		schemas, diags = c.MaybeGetSchemas(srcStateFile.State, nil)
	}

	if err := stateMgr.WriteState(srcStateFile.State); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to write state: %s", err))
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to persist state: %s", err))
		return 1
	}

	view.Diagnostics(diags)
	return 0
}

func (c *StatePushCommand) Help() string {
	helpText := `
Usage: terraform [global options] state push [options] PATH

  Update remote state from a local state file at PATH.

  This command "pushes" a local state and overwrites remote state
  with a local state file. The command will protect you against writing
  an older serial or a different state file lineage unless you specify the
  "-force" flag.

  This command works with local state (it will overwrite the local
  state), but is less useful for this use case.

  If PATH is "-", then this command will read the state to push from stdin.
  Data from stdin is not streamed to the backend: it is loaded completely
  (until pipe close), verified, and then pushed.

Options:

  -force              Write the state even if lineages don't match or the
                      remote serial is higher.

  -lock=false         Don't hold a state lock during the operation. This is
                      dangerous if others might concurrently run commands
                      against the same workspace.

  -lock-timeout=0s    Duration to retry a state lock.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePushCommand) Synopsis() string {
	return "Update remote state from a local state file"
}
