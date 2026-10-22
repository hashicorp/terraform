// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphCommand is a Command implementation that takes a Terraform
// configuration and outputs the dependency tree in graphical form.
type GraphCommand struct {
	Meta
}

func (c *GraphCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	args, argDiags := arguments.ParseGraph(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewGraph(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get the root module directory
	configPath := c.Meta.WorkingDir.RootModuleDir()

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("Error loading plugin path: %w", err))
		view.Diagnostics(diags)
		return 1
	}

	// Try to load plan if path is specified
	var planFile *planfile.WrappedPlanFile
	if args.Plan != "" {
		planFile, err = c.PlanFile(args.Plan)
		if err != nil {
			diags = diags.Append(err)
			view.Diagnostics(diags)
			return 1
		}
	}

	// Load the backend
	b, backendDiags := c.backend(".", arguments.ViewHuman)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backendrun.Local)
	if !ok {
		diags = diags.Append(errors.New(ErrUnsupportedLocalOp))
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Build the operation
	opReq := c.Operation(b, arguments.ViewHuman)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
	opReq.PlanFile = planFile
	opReq.AllowUnsetVariables = true
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	var varDiags tfdiags.Diagnostics
	opReq.Variables, varDiags = args.Vars.CollectValues(func(filename string, src []byte) {
		opReq.ConfigLoader.Parser().ForceFileSource(filename, src)
	})
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get the context
	lr, _, ctxDiags := local.LocalRun(context.Background(), opReq)

	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	lr.Core.SetGraphOpts(&terraform.ContextGraphOpts{SkipGraphValidation: args.DrawCycles})

	if args.GraphType == "" {
		if planFile == nil {
			// Simple resource dependency mode:
			// This is based on the plan graph but we then further reduce it down
			// to just resource dependency relationships, assuming that in most
			// cases the most important thing is what order we'll visit the
			// resources in.
			fullG, graphDiags := lr.Core.PlanGraphForUI(lr.Config, lr.InputState, plans.NormalMode)
			diags = diags.Append(graphDiags)
			if graphDiags.HasErrors() {
				view.Diagnostics(diags)
				return 1
			}

			g := fullG.ResourceGraph()
			view.ResourceOnlyGraph(g)
			return 0
		} else {
			args.GraphType = "apply"
		}
	}

	var g *terraform.Graph
	var graphDiags tfdiags.Diagnostics
	switch args.GraphType {
	case "plan":
		g, graphDiags = lr.Core.PlanGraphForUI(lr.Config, lr.InputState, plans.NormalMode)
	case "plan-refresh-only":
		g, graphDiags = lr.Core.PlanGraphForUI(lr.Config, lr.InputState, plans.RefreshOnlyMode)
	case "plan-destroy":
		g, graphDiags = lr.Core.PlanGraphForUI(lr.Config, lr.InputState, plans.DestroyMode)
	case "apply":
		plan := lr.Plan

		// Historically "terraform graph" would allow the nonsensical request to
		// render an apply graph without a plan, so we continue to support that
		// here, though perhaps one day this should be an error.
		if lr.Plan == nil {
			plan = &plans.Plan{
				Changes:      plans.NewChangesSrc(),
				UIMode:       plans.NormalMode,
				PriorState:   lr.InputState,
				PrevRunState: lr.InputState,
			}
		}

		g, graphDiags = lr.Core.ApplyGraphForUI(plan, lr.Config)
	case "eval", "validate":
		// Terraform v0.12 through v1.0 supported both of these, but the
		// graph variants for "eval" and "validate" are purely implementation
		// details and don't reveal anything (user-model-wise) that you can't
		// see in the plan graph.
		graphDiags = graphDiags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Graph type no longer available",
			fmt.Sprintf("The graph type %q is no longer available. Use -type=plan instead to get a similar result.", args.GraphType),
		))
	default:
		graphDiags = graphDiags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported graph type",
			`The -type=... argument must be either "plan", "plan-refresh-only", "plan-destroy", or "apply".`,
		))
	}
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	var graphStr string

	opts := &dag.DotOpts{
		DrawCycles: args.DrawCycles,
		MaxDepth:   args.ModuleDepth,
		Verbose:    args.Verbose,
	}

	switch args.ViewType {
	case arguments.ViewMermaidGraph:
		graphStr, err = terraform.GraphMermaid(g, opts)
	case arguments.ViewDotGraph:
		graphStr, err = terraform.GraphDot(g, opts)
	default:
		diags = diags.Append(fmt.Errorf("Unsupported graph format: %s", args.ViewType))
		view.Diagnostics(diags)
		return 1
	}
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error converting graph: %w", err))
		view.Diagnostics(diags)
		return 1
	}

	if diags.HasErrors() {
		// For this command we only show diagnostics if there are errors,
		// because printing out naked warnings could upset a naive program
		// consuming our graph output.
		view.Diagnostics(diags)
		return 1
	}

	view.Graph(graphStr)
	return 0
}

func (c *GraphCommand) Help() string {
	helpText := `
Usage: terraform [global options] graph [options]

  Produces a representation of the dependency graph between different
  objects in the current configuration and state.

  By default the graph shows a summary only of the relationships between
  resources in the configuration, since those are the main objects that
  have side-effects whose ordering is significant. You can generate more
  detailed graphs reflecting Terraform's actual evaluation strategy
  by specifying the -type=TYPE option to select an operation type.

  The graph is presented in the DOT language. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.

Options:

  -compact-warnings    If Terraform produces any warnings that are not
                       accompanied by errors, show them in a more compact
                       form that includes only the summary messages

  -draw-cycles         Highlight any cycles in the graph with colored edges.
                       This helps when diagnosing cycle errors. This option is
                       supported only when illustrating a real evaluation graph,
                       selected using the -type=TYPE option.

  -format=FORMAT       Output format for the graph. Supported values are
                       dot (default) and mermaid.

  -module-depth=n      (deprecated) In prior versions of Terraform, specified the
                       depth of modules to show in the output.

  -no-color            If specified, non-graph output won't contain any color.

  -plan=tfplan         Render graph using the specified plan file instead of the
                       configuration in the current directory. Implies -type=apply.

  -type=TYPE           Type of operation graph to output. Can be: plan,
                       plan-refresh-only, plan-destroy, or apply. By default
                       Terraform just summarizes the relationships between the
                       resources in your configuration, without any particular
                       operation in mind. Full operation graphs are more detailed
                       but therefore often harder to read.

  -var 'foo=bar'       Set a value for one of the input variables in the root
                       module of the configuration. Use this option more than
                       once to set more than one variable.

  -var-file=filename   Load variable values from the given file, in addition
                       to the default files terraform.tfvars and *.auto.tfvars.
                       Use this option more than once to include more than one
                       variables file.
`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Generate a Graphviz graph of the steps in an operation"
}
