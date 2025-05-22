// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

var _ terraform.GraphTransformer = (*EvalContextTransformer)(nil)

// EvalContextTransformer should be the first node to execute in the graph, and
// it initialises the run blocks and state files in the evaluation context.
// TODO(liamcervante): Also initialise the variables in here when needed.
type EvalContextTransformer struct {
	File *moduletest.File
}

func (e *EvalContextTransformer) Transform(graph *terraform.Graph) error {
	node := &dynamicNode{
		eval: func(ctx *EvalContext) {
			for _, run := range e.File.Runs {

				// Within the run outputs a nil but present entry means the
				// run block exists but hasn't executed yet.
				// TODO(liamcervante): Once providers are embedded in the graph
				// we don't need to track run blocks in this way anymore.

				ctx.SetOutput(run, cty.NilVal)

				// We also want to set an empty state file for every state key
				// we're going to be executing within the graph.

				key := run.GetStateKey()
				if state := ctx.GetFileState(key); state == nil {
					ctx.SetFileState(key, &TestFileState{
						Run:   nil,
						State: states.NewState(),
					})
				}

			}
		},
	}

	graph.Add(node)
	for _, v := range graph.Vertices() {
		if v == node {
			continue
		}
		graph.Connect(dag.BasicEdge(v, node))
	}

	return nil
}
