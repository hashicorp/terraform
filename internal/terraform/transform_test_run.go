// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs to the graph.
type TestRunTransformer struct {
	File *moduletest.File
}

func (t *TestRunTransformer) Transform(g *Graph) error {
	prevs := make(map[string]*NodeTestRun)
	for _, run := range t.File.Runs {
		node := &NodeTestRun{run: run, file: t.File}
		g.Add(node)
		refs, _ := run.GetReferences()
		for _, ref := range refs {
			subjectStr := ref.Subject.String()
			if !strings.HasPrefix(subjectStr, "run.") {
				continue
			}
			runName := strings.TrimPrefix(subjectStr, "run.")
			if runName == "" {
				continue
			}
			dependency, ok := prevs[runName]
			if !ok {
				// TODO: should we catch this error, or leave it, as it will still be caught in the test?
				return fmt.Errorf("dependency %q not found for run %q", runName, run.Name)
			}
			g.Connect(dag.BasicEdge(node, dependency))
		}
		prevs[run.Name] = node
	}

	return nil
}

// -------------------------------------------------------- CloseTestRootModuleTransformer --------------------------------------------------------

// CloseTestRootModuleTransformer is a GraphTransformer that adds a root to the graph.
type CloseTestRootModuleTransformer struct{}

func (t *CloseTestRootModuleTransformer) Transform(g *Graph) error {
	// close the root module
	closeRoot := &nodeCloseTest{}
	g.Add(closeRoot)

	// since this is closing the root module, make it depend on everything in
	// the root module.
	for _, v := range g.Vertices() {
		if v == closeRoot {
			continue
		}

		// since this is closing the root module,  and must be last, we can
		// connect to anything that doesn't have any up edges.
		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(closeRoot, v))
		}
	}

	return nil
}

// This node doesn't do anything, it's just to ensure that we have a single
// root node that depends on everything in the root module.
type nodeCloseTest struct {
}

// -------------------------------------------------------- ApplyNoParallelTransformer --------------------------------------------------------

// ApplyNoParallelTransformer ensures that all apply operations are run in sequential order.
type ApplyNoParallelTransformer struct{}

func (t *ApplyNoParallelTransformer) Transform(g *Graph) error {
	// find all the apply nodes
	runs := make([]*NodeTestRun, 0)
	for _, v := range g.Vertices() {
		run, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		if run.run.Config.Command == configs.ApplyTestCommand {
			runs = append(runs, run)
		}
	}

	// sort them in descending order
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].run.Index > runs[j].run.Index
	})

	// connect them all in serial
	for i := 0; i < len(runs)-1; i++ {
		g.Connect(dag.BasicEdge(runs[i], runs[i+1]))
	}

	return nil
}

// -------------------------------------------------------- TestFileVariableTransformer --------------------------------------------------------

// TestFileVariableTransformer is a GraphTransformer that adds variables from a test file to the graph.
type TestFileVariableTransformer struct {
	File *moduletest.File
}

func (t *TestFileVariableTransformer) Transform(g *Graph) error {
	// for _, expr := range t.File.Config.Variables {

	// }

	return nil
}
