// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

// Graph represents the command-line arguments for the graph command.
type Graph struct {
	DrawCycles  bool
	GraphType   string
	ModuleDepth int
	Verbose     bool
	PlanPath    string
	Args        []string
}

// ParseGraph processes CLI arguments, returning a Graph value and errors.
// If errors are encountered, a Graph value is still returned representing
// the best effort interpretation of the arguments.
func ParseGraph(args []string) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	graph := &Graph{}

	cmdFlags := defaultFlagSet("graph")
	cmdFlags.BoolVar(&graph.DrawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graph.GraphType, "type", "", "type")
	cmdFlags.IntVar(&graph.ModuleDepth, "module-depth", -1, "module-depth")
	cmdFlags.BoolVar(&graph.Verbose, "verbose", false, "verbose")
	cmdFlags.StringVar(&graph.PlanPath, "plan", "", "plan")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	graph.Args = cmdFlags.Args()

	return graph, diags
}
