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

// List is used to assemble the list of Workspaces and log it via Output
func (v *workspaceHuman) List(selected string, list []string, diags tfdiags.Diagnostics) {
	// Print diags above output
	v.meta.showDiagnostics(diags)

	// Print list
	if len(list) > 0 {
		var out bytes.Buffer
		for _, s := range list {
			if s == selected {
				out.WriteString("* ")
			} else {
				out.WriteString("  ")
			}
			out.WriteString(s + "\n")
		}
		v.ui.Output(out.String())
	}
}

// newWorkspace returns a views.Workspace interface.
//
// When human-readable output is migrated from cli.Ui to views.View this method should be deleted and
// replaced with using views.NewWorkspace directly.
func newWorkspaceList(vt arguments.ViewType, view *views.View, ui cli.Ui, meta *Meta) views.WorkspaceList {
	switch vt {
	case arguments.ViewJSON:
		return views.NewWorkspaceList(vt, view)
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
	view := newWorkspaceList(viewType, c.View, c.Ui, &c.Meta)
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
		view.List("", nil, diags)
		return 1
	}

	// Load the backend
	b, diags := c.backend(configPath, viewType)
	if diags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	// This command will not write state
	c.ignoreRemoteVersionConflict(b)

	states, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	env, isOverridden := c.WorkspaceOverridden()

	if isOverridden {
		warn := tfdiags.Sourceless(
			tfdiags.Warning,
			envIsOverriddenNote,
			"",
		)
		diags = diags.Append(warn)
	}

	// Print:
	// 1. Diagnostics
	// 2. The list of workspaces, highlighting the current workspace
	view.List(env, states, diags)

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
