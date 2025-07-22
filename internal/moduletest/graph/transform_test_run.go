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
	skip bool // If true, the transformer will skip adding run nodes to the graph.
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	if t.skip {
		return nil
	}
	// Create and add nodes for each run
	var nodes []*NodeTestRun
	for _, run := range t.opts.File.Runs {
		node := &NodeTestRun{run: run, opts: t.opts, priorRuns: make(map[string]*moduletest.Run)}
		g.Add(node)
		nodes = append(nodes, node)
	}

	// Connect nodes based on dependencies
	ControlParallelism(g, nodes)

	// Runs with the same state key inherently depend on each other, so we
	// connect them sequentially.
	t.connectSameStateRuns(g, nodes)

	return nil
}

func (t *TestRunTransformer) connectSameStateRuns(g *terraform.Graph, nodes []*NodeTestRun) {
	stateRuns := make(map[string][]*NodeTestRun)
	for _, node := range nodes {
		key := node.run.Config.StateKey
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
func ControlParallelism[T any](g *terraform.Graph, nodes []T) {
	for i, node := range nodes {
		switch node := any(node).(type) {
		case *NodeTestRun:
			if node.run.Config.Parallel {
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
