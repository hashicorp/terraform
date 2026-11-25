// Copyright IBM Corp. 2014, 2026
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
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StatePushCommand is a Command implementation that allows
// pushing a local state to a remote location.
type StatePushCommand struct {
	Meta
	StateMeta
}

func (c *StatePushCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	parsedArgs, parseDiags := arguments.ParseStatePush(rawArgs)
	diags = diags.Append(parseDiags)

	// Prepare the view
	view := views.NewStatePush(parsedArgs.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	c.Meta.stateLock = parsedArgs.StateLock
	c.Meta.stateLockTimeout = parsedArgs.StateLockTimeout
	c.Meta.ignoreRemoteVersion = parsedArgs.IgnoreRemoteVersion

	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	var varDiags tfdiags.Diagnostics
	c.VariableValues, varDiags = parsedArgs.Vars.CollectValues(func(filename string, src []byte) {
		loader.Parser().ForceFileSource(filename, src)
	})
	diags = diags.Append(varDiags)
	if varDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	vDiags := c.Meta.checkRequiredVersion()
	diags = diags.Append(vDiags)
	if vDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Determine our reader for the input state. This is the filepath
	// or stdin if "-" is given.
	var r io.Reader = os.Stdin
	if parsedArgs.Path != "-" {
		f, err := os.Open(parsedArgs.Path)
		if err != nil {
			diags = diags.Append(
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to open state",
					fmt.Sprintf("Error opening state %q: %s", parsedArgs.Path, err),
				),
			)
			view.Diagnostics(diags)
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
		diags = diags.Append(
			tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read source state",
				fmt.Sprintf("Error reading source state %q: %s", parsedArgs.Path, err),
			),
		)
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, diags := c.backend(".", parsedArgs.ViewType)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Determine the workspace name
	workspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to select workspace",
			fmt.Sprintf("Error selecting workspace: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, workspace)
	diags = diags.Append(remoteVersionDiags)
	if remoteVersionDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get the state manager for the currently-selected workspace
	stateMgr, sDiags := b.StateMgr(workspace)
	if sDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to access state",
			fmt.Sprintf("Failed to access destination state: %s", sDiags.Err()),
		))
		view.Diagnostics(diags)
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(parsedArgs.ViewType, c.View))
		if lockDiags := stateLocker.Lock(stateMgr, "state-push"); lockDiags.HasErrors() {
			diags = diags.Append(lockDiags)
			view.Diagnostics(diags)
			return 1
		}
		defer func() {
			if unlockDiags := stateLocker.Unlock(); unlockDiags.HasErrors() {
				diags = diags.Append(unlockDiags)
				view.Diagnostics(diags)
			}
		}()
	}

	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to refresh the state",
			fmt.Sprintf("Error refreshing the destination state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	if srcStateFile == nil {
		// We'll push a new empty state instead
		srcStateFile = statemgr.NewStateFile()
	}

	// Import it, forcing through the lineage/serial if requested and possible.
	if err := statemgr.Import(srcStateFile, stateMgr, parsedArgs.Force); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to save the state",
			fmt.Sprintf("Terraform encountered an error when saving state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Get schemas, if possible, before writing state
	var schemas *terraform.Schemas
	if isCloudMode(b) {
		schemas, diags = c.MaybeGetSchemas(srcStateFile.State, nil)
	}

	if err := stateMgr.WriteState(srcStateFile.State); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to save the state",
			fmt.Sprintf("Terraform encountered an error when saving state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.PersistState(schemas); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to save the state",
			fmt.Sprintf("Terraform encountered an error when saving state: %s", err),
		))
		view.Diagnostics(diags)
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

  -compact-warnings   If Terraform produces any warnings that are not
                      accompanied by errors, show them in a more compact
                      form that includes only the summary messages

  -force              Write the state even if lineages don't match or the
                      remote serial is higher.

  -lock=false         Don't hold a state lock during the operation. This is
                      dangerous if others might concurrently run commands
                      against the same workspace.

  -lock-timeout=0s    Duration to retry a state lock.

  -no-color           Disable text coloring in the output.

  -var 'foo=bar'      Set a value for one of the input variables in the root
                      module of the configuration. Use this option more than
                      once to set more than one variable.

  -var-file=filename  Load variable values from the given file, in addition
                      to the default files terraform.tfvars and *.auto.tfvars.
                      Use this option more than once to include more than one
                      variables file.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePushCommand) Synopsis() string {
	return "Update remote state from a local state file"
}
