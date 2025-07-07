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
			if target := t.opts.File.Runs[ix]; !run.Config.Parallel || !target.Config.Parallel || run.Config.StateKey == target.Config.StateKey {
				priorRuns[target.Name] = target
			}
		}

		g.Add(&NodeTestRun{
			run:       run,
			opts:      t.opts,
			priorRuns: priorRuns,
		})
	}

	// Connect nodes based on dependencies
	ControlParallelism(g, nodes, t.opts.DebugMode)

	// Runs with the same state key inherently depend on each other, so we
	// connect them sequentially.
	t.connectSameStateRuns(g, nodes)

	return nil
}

func (t *TestRunTransformer) connectSameStateRuns(g *terraform.Graph, nodes []*NodeTestRun) {
	stateRuns := make(map[string][]*NodeTestRun)
	for _, node := range nodes {
		key := node.run.GetStateKey()
		stateRuns[key] = append(stateRuns[key], node)
	}
	for _, runs := range stateRuns {
		for i := 1; i < len(runs); i++ {
			curr, prev := runs[i], runs[i-1]
			curr.priorRuns[prev.run.Name] = prev.run
			g.Connect(dag.BasicEdge(curr, prev))
		}
	}
}

// ControlParallelism connects nodes in the graph based on their parallelism
// settings. If a node opts out of parallelism, it will be connected sequentially
// to all previous and subsequent nodes that are also part of the parallelism
// control flow.
func ControlParallelism[T any](g *terraform.Graph, nodes []T, debugMode bool) {
	for i, node := range nodes {
		switch node := any(node).(type) {
		case *NodeTestRun:
			// If a node has a breakpoint set, it will not connect to
			// any runs, allowing it to run independently.
			// TODO: If debug mode does not run tests sequentially, functions like
			// `next` will be non-deterministic.
			if node.run.Config.Parallel && !debugMode {
				continue
			}

			for j := range i {
				refNode := any(nodes[j]).(*NodeTestRun)
				node.priorRuns[refNode.run.Name] = refNode.run
			}
		case *NodeStateCleanup:
			if node.parallel {
				continue
			}
		default:
			// If the node type does not support parallelism, skip it.
			continue
		}

		// Connect to all previous runs
		for j := range i {
			g.Connect(dag.BasicEdge(node, nodes[j]))
		}

		// Connect to all subsequent runs
		for j := i + 1; j < len(nodes); j++ {
			g.Connect(dag.BasicEdge(nodes[j], node))
		}
	}
}
