// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestRunTransformer struct {
	opts *graphOptions
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	// Create and add nodes for each run
	for _, run := range t.opts.File.Runs {
		priorRuns := make(map[string]*moduletest.Run)
		for ix := run.Index - 1; ix >= 0; ix-- {
			// If either node isn't parallel, we should draw an edge between
			// them. Also, if they share the same state key we should also draw
			// an edge between them regardless of the parallelisation.
			// If debug mode is enabled, we should draw an edge between them so that nodes are not executed in parallel.
			if target := t.opts.File.Runs[ix]; !run.Config.Parallel || !target.Config.Parallel || run.Config.StateKey == target.Config.StateKey || t.opts.DebugMode {
				priorRuns[target.Name] = target
			}
		}

		g.Add(&NodeTestRun{
			run:       run,
			opts:      t.opts,
			priorRuns: priorRuns,
		})
	}

	return nil
}
