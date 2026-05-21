// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Graph represents the command-line arguments for the graph command.
type Graph struct {
	// DrawCycles highlights any cycles in the graph with colored edges.
	DrawCycles bool

	// GraphType is the type of operation graph to output (plan,
	// plan-refresh-only, plan-destroy, or apply). Empty string means the
	// default resource-dependency summary.
	GraphType string

	// ModuleDepth is a deprecated option that was used in prior versions to
	// control the depth of modules shown.
	ModuleDepth int

	// Verbose enables verbose graph output.
	Verbose bool

	// Plan is the path to a saved plan file to render as a graph.
	Plan string

	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars
}

// ParseGraph processes CLI arguments, returning a Graph value and errors.
// If errors are encountered, a Graph value is still returned representing
// the best effort interpretation of the arguments.
func ParseGraph(args []string) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	graph := &Graph{
		ModuleDepth: -1,
		Vars:        &Vars{},
	}

	cmdFlags := extendedFlagSet("graph", nil, nil, graph.Vars)
	cmdFlags.BoolVar(&graph.DrawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graph.GraphType, "type", "", "type")
	cmdFlags.IntVar(&graph.ModuleDepth, "module-depth", -1, "module-depth")
	cmdFlags.BoolVar(&graph.Verbose, "verbose", false, "verbose")
	cmdFlags.StringVar(&graph.Plan, "plan", "", "plan")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected no positional arguments. Did you mean to use -chdir?",
		))
	}

	return graph, diags
}
