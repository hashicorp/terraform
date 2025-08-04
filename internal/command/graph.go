// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
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

func (c *GraphCommand) Run(args []string) int {
	var drawCycles bool
	var graphTypeStr string
	var moduleDepth int
	var verbose bool
	var planPath string

	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("graph")
	cmdFlags.BoolVar(&drawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graphTypeStr, "type", "", "type")
	cmdFlags.IntVar(&moduleDepth, "module-depth", -1, "module-depth")
	cmdFlags.BoolVar(&verbose, "verbose", false, "verbose")
	cmdFlags.StringVar(&planPath, "plan", "", "plan")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Try to load plan if path is specified
	var planFile *planfile.WrappedPlanFile
	if planPath != "" {
		planFile, err = c.PlanFile(planPath)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backendrun.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
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
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	lr, _, ctxDiags := local.LocalRun(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}
	lr.Core.SetGraphOpts(&terraform.ContextGraphOpts{SkipGraphValidation: drawCycles})

	if graphTypeStr == "" {
		if planFile == nil {
			// Simple resource dependency mode:
			// This is based on the plan graph but we then further reduce it down
			// to just resource dependency relationships, assuming that in most
			// cases the most important thing is what order we'll visit the
			// resources in.
			fullG, graphDiags := lr.Core.PlanGraphForUI(lr.Config, lr.InputState, plans.NormalMode)
			diags = diags.Append(graphDiags)
			if graphDiags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}

			g := fullG.ResourceGraph()
			return c.resourceOnlyGraph(g)
		} else {
			graphTypeStr = "apply"
		}
	}

	var g *terraform.Graph
	var graphDiags tfdiags.Diagnostics
	switch graphTypeStr {
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
			fmt.Sprintf("The graph type %q is no longer available. Use -type=plan instead to get a similar result.", graphTypeStr),
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
		c.showDiagnostics(diags)
		return 1
	}

	graphStr, err := terraform.GraphDot(g, &dag.DotOpts{
		DrawCycles: drawCycles,
		MaxDepth:   moduleDepth,
		Verbose:    verbose,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error converting graph: %s", err))
		return 1
	}

	if diags.HasErrors() {
		// For this command we only show diagnostics if there are errors,
		// because printing out naked warnings could upset a naive program
		// consuming our dot output.
		c.showDiagnostics(diags)
		return 1
	}

	_, err = c.Streams.Stdout.File.WriteString(graphStr)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to write graph to stdout: %s", err))
		return 1
	}

	return 0
}

func (c *GraphCommand) resourceOnlyGraph(graph addrs.DirectedGraph[addrs.ConfigResource]) int {
	out := c.Streams.Stdout.File
	fmt.Fprintln(out, "digraph G {")
	// Horizontal presentation is easier to read because our nodes tend
	// to be much wider than they are tall. The leftmost nodes in the output
	// are those Terraform would visit first.
	fmt.Fprintln(out, "  rankdir = \"RL\";")
	fmt.Fprintln(out, "  node [shape = rect, fontname = \"sans-serif\"];")

	// To help relate the output back to the configuration it came from,
	// and to make the individual node labels more reasonably sized when
	// deeply nested inside modules, we'll cluster the nodes together by
	// the module they belong to and then show only the local resource
	// address in the individual nodes. We'll accomplish that by sorting
	// the nodes first by module, so we can then notice the transitions.
	allAddrs := graph.AllNodes()
	if len(allAddrs) == 0 {
		fmt.Fprintln(out, "  /* This configuration does not contain any resources.         */")
		fmt.Fprintln(out, "  /* For a more detailed graph, try: terraform graph -type=plan */")
	}
	addrsOrder := make([]addrs.ConfigResource, 0, len(allAddrs))
	for _, addr := range allAddrs {
		addrsOrder = append(addrsOrder, addr)
	}
	sort.Slice(addrsOrder, func(i, j int) bool {
		iAddr, jAddr := addrsOrder[i], addrsOrder[j]
		iModStr, jModStr := iAddr.Module.String(), jAddr.Module.String()
		switch {
		case iModStr != jModStr:
			return iModStr < jModStr
		default:
			iRes, jRes := iAddr.Resource, jAddr.Resource
			switch {
			case iRes.Mode != jRes.Mode:
				return iRes.Mode == addrs.DataResourceMode
			case iRes.Type != jRes.Type:
				return iRes.Type < jRes.Type
			default:
				return iRes.Name < jRes.Name
			}
		}
	})

	currentMod := addrs.RootModule
	for _, addr := range addrsOrder {
		if !addr.Module.Equal(currentMod) {
			// We need a new subgraph, then.
			// Experimentally it seems like nested clusters tend to make it
			// hard for dot to converge on a good layout, so we'll stick with
			// just one level of clusters for now but could revise later based
			// on feedback.
			if !currentMod.IsRoot() {
				fmt.Fprintln(out, "  }")
			}
			currentMod = addr.Module
			fmt.Fprintf(out, "  subgraph \"cluster_%s\" {\n", currentMod.String())
			fmt.Fprintf(out, "    label = %q\n", currentMod.String())
			fmt.Fprintf(out, "    fontname = %q\n", "sans-serif")
		}
		if currentMod.IsRoot() {
			fmt.Fprintf(out, "  %q [label=%q];\n", addr.String(), addr.Resource.String())
		} else {
			fmt.Fprintf(out, "    %q [label=%q];\n", addr.String(), addr.Resource.String())
		}
	}
	if !currentMod.IsRoot() {
		fmt.Fprintln(out, "  }")
	}

	// Now we'll emit all of the edges.
	// We use addrsOrder for both levels to ensure a consistent ordering between
	// runs without further sorting, which means we visit more nodes than we
	// really need to but this output format is only really useful for relatively
	// small graphs anyway, so this should be fine.
	for _, sourceAddr := range addrsOrder {
		deps := graph.DirectDependenciesOf(sourceAddr)
		for _, targetAddr := range addrsOrder {
			if !deps.Has(targetAddr) {
				continue
			}
			fmt.Fprintf(out, "  %q -> %q;\n", sourceAddr.String(), targetAddr.String())
		}
	}

	fmt.Fprintln(out, "}")
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

  -plan=tfplan     Render graph using the specified plan file instead of the
                   configuration in the current directory. Implies -type=apply.

  -draw-cycles     Highlight any cycles in the graph with colored edges.
                   This helps when diagnosing cycle errors. This option is
                   supported only when illustrating a real evaluation graph,
                   selected using the -type=TYPE option.

  -type=TYPE       Type of operation graph to output. Can be: plan,
                   plan-refresh-only, plan-destroy, or apply. By default
                   Terraform just summarizes the relationships between the
                   resources in your configuration, without any particular
                   operation in mind. Full operation graphs are more detailed
                   but therefore often harder to read.

  -module-depth=n  (deprecated) In prior versions of Terraform, specified the
                   depth of modules to show in the output.
`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Generate a Graphviz graph of the steps in an operation"
}
