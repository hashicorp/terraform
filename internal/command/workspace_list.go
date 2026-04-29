// Copyright IBM Corp. 2014, 2026
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

func (c *WorkspaceListCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// c.Meta.process removes global flags (-no-color, -compact-warnings) and uses them to configure the Ui and View.
	//
	// Other command implementations remove those arguments via arguments.ParseView, instead. That is only possible if views
	// are used for both human and machine output. This command still uses cli.Ui for human output, so c.Meta.process is necessary.
	rawArgs = c.Meta.process(rawArgs)

	// Parse command-specific arguments.
	args, diags := arguments.ParseWorkspaceList(rawArgs)

	// Prepare the view
	//
	// Note - here the view uses:
	// - cli.Ui for human output
	// - view.View for machine-readable output
	//
	// Note: We don't call c.View.Configure here after obtaining the view because it's already called in c.Meta.process.
	// TODO: When we migrate human output to use views fully instead of cli.Ui we would replace using c.Meta.process with arguments.ParseView.
	// arguments.ParseView returns a 'common' View that can be used as an argument in the c.View.Configure method.
	view := newWorkspaceList(args.ViewType, c.View, c.Ui, &c.Meta)

	// Warn against using `terraform env` commands
	if args.ViewType == arguments.ViewHuman {
		envCommandShowWarning(c.Ui, c.LegacyName)
	} else {
		diags = diags.Append(envCommandWarningDiag(c.LegacyName))
	}

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.List("", nil, diags)
		return 1
	}

	// Load the backend
	configPath := c.WorkingDir.RootModuleDir()
	b, bDiags := c.backend(configPath, args.ViewType)
	diags = diags.Append(bDiags)
	if bDiags.HasErrors() {
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

Options:

  -json            If specified, machine readable output will be
                   printed in JSON format.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}

type workspaceListHuman struct {
	ui   cli.Ui
	meta *Meta
}

// List is used to assemble the list of Workspaces and log it via Output
func (v *workspaceListHuman) List(selected string, list []string, diags tfdiags.Diagnostics) {
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
	} else {
		// Warn that no states exist
		v.meta.showDiagnostics(warnNoEnvsExistDiag(selected))
	}
}

// newWorkspaceList returns a views.WorkspaceList interface.
//
// When human-readable output is migrated from cli.Ui to views.View this method should be deleted and
// replaced with using a views.NewWorkspaceList method.
func newWorkspaceList(vt arguments.ViewType, view *views.View, ui cli.Ui, meta *Meta) views.WorkspaceList {
	switch vt {
	case arguments.ViewJSON:
		return views.NewWorkspaceList(vt, view)
	case arguments.ViewHuman:
		return &workspaceListHuman{
			ui:   ui,
			meta: meta,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}
