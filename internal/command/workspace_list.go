// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

type workspaceHuman struct {
	ui   cli.Ui
	meta *Meta
}

// Diagnostics renders diagnostics using old-style logic that sends:
// Error diagnostics to stderr via ui.Error
// Warning diagnostics to stderr via ui.Warn
// Anything else to stdout via ui.Output
func (v *workspaceHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.meta.showDiagnostics(diags)
}

// Output is used to render text in the terminal, via stdout
func (v *workspaceHuman) Output(msg string) {
	v.ui.Output(msg)
}

// Warn is used to render warning text in the terminal, via stderr
//
// This is here for backwards compatibility reasons.
// In future calling code should use Diagnostics directly.
func (v *workspaceHuman) Warn(msg string) {
	v.ui.Warn(msg)
}

// Error is used to render error text in the terminal, via stderr
//
// This is here for backwards compatibility reasons.
// In future calling code should use Diagnostics directly.
func (v *workspaceHuman) Error(msg string) {
	v.ui.Error(msg)
}

// newWorkspace returns a views.Workspace interface.
//
// When human-readable output is migrated from cli.Ui to views.View this method should be deleted and
// replaced with using views.NewWorkspace directly.
func newWorkspace(vt arguments.ViewType, view *views.View, ui cli.Ui, meta *Meta) views.Workspace {
	switch vt {
	case arguments.ViewJSON:
		return views.NewWorkspace(vt, view)
	case arguments.ViewHuman:
		return &workspaceHuman{
			ui:   ui,
			meta: meta,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

func (c *WorkspaceListCommand) Run(args []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments, e.g. -no-color
	common, args := arguments.ParseView(args)
	// Propagate -no-color for legacy use of Ui.  The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	var jsonOutput bool
	cmdFlags := c.Meta.defaultFlagSet("workspace list")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	// Prepare the view
	//
	// Note - here the view uses:
	// - cli.Ui for human output
	// - view.View for machine-readable output
	var viewType arguments.ViewType
	if jsonOutput {
		viewType = arguments.ViewJSON
	} else {
		viewType = arguments.ViewHuman
	}
	view := newWorkspace(viewType, c.View, c.Ui, &c.Meta)
	c.View.Configure(common)

	// Warn against using `terraform env` commands
	if jsonOutput {
		envCommandShowWarningWithView(c.View, c.LegacyName)
	} else {
		envCommandShowWarning(c.Ui, c.LegacyName)
	}

	args = cmdFlags.Args()
	configPath, err := ModulePath(args)
	if err != nil {
		diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, diags := c.backend(configPath, viewType)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	view.Diagnostics(diags) // output warnings, if any

	env, isOverridden := c.WorkspaceOverridden()

	var out bytes.Buffer
	for _, s := range states {
		if s == env {
			out.WriteString("* ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(s + "\n")
	}

	view.Output(out.String())

	if isOverridden {
		view.Output(envIsOverriddenNote)
	}

	return 0
}

func (c *WorkspaceListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("")
}

func (c *WorkspaceListCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceListCommand) Help() string {
	helpText := `
Usage: terraform [global options] workspace list

  List Terraform workspaces.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}
