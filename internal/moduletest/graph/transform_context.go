// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

var _ terraform.GraphTransformer = (*EvalContextTransformer)(nil)

// EvalContextTransformer should be the first node to execute in the graph, and
// it initialises the run blocks and state files in the evaluation context.
type EvalContextTransformer struct {
	File *moduletest.File
}

func (e *EvalContextTransformer) Transform(graph *terraform.Graph) error {
	node := &dynamicNode{
		eval: func(ctx *EvalContext) {
			for _, run := range e.File.Runs {
				// initialise all the state keys before the graph starts
				// properly
				key := run.Config.StateKey
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
	for v := range graph.VerticesSeq() {
		if v == node {
			continue
		}
		graph.Connect(dag.BasicEdge(v, node))
	}

	return nil
}
