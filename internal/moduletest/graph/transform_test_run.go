// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/dag"
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
	var nodes []*NodeTestRun
	for _, run := range t.opts.File.Runs {
		node := &NodeTestRun{run: run, opts: t.opts}
		g.Add(node)
		nodes = append(nodes, node)
	}

	// Connect nodes based on dependencies
	t.controlParallelism(g, nodes)

	// Runs with the same state key inherently depend on each other, so we
	// connect them sequentially.
	t.connectSameStateRuns(g, nodes)

	return nil
}

func (t *TestRunTransformer) controlParallelism(g *terraform.Graph, nodes []*NodeTestRun) {

	// If there is a run that has opted out of parallelism, we will connect it
	// sequentially to all previous and subsequent runs. This effectively
	// divides the parallelizable runs into separate groups, ensuring that
	// non-parallelizable runs are executed in sequence with respect to all
	// other runs.

	for i, node := range nodes {
		if node.run.Config.Parallel {
			continue
		}

		// Connect to all previous runs
		for j := 0; j < i; j++ {
			g.Connect(dag.BasicEdge(node, nodes[j]))
		}

		// Connect to all subsequent runs
		for j := i + 1; j < len(nodes); j++ {
			g.Connect(dag.BasicEdge(nodes[j], node))
		}
	}
}

func (t *TestRunTransformer) connectSameStateRuns(g *terraform.Graph, nodes []*NodeTestRun) {
	stateRuns := make(map[string][]*NodeTestRun)
	for _, node := range nodes {
		key := node.run.GetStateKey()
		stateRuns[key] = append(stateRuns[key], node)
	}
	for _, runs := range stateRuns {
		for i := 1; i < len(runs); i++ {
			g.Connect(dag.BasicEdge(runs[i], runs[i-1]))
		}
	}
}

func (t *TestRunTransformer) getVariableNames(run *moduletest.Run) map[string]struct{} {
	set := make(map[string]struct{})
	for name := range t.opts.GlobalVars {
		set[name] = struct{}{}
	}
	for name := range run.Config.Variables {
		set[name] = struct{}{}
	}

	for name := range t.opts.File.Config.Variables {
		set[name] = struct{}{}
	}
	for name := range run.ModuleConfig.Module.Variables {
		set[name] = struct{}{}
	}
	return set
}
