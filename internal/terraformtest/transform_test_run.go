// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestRunTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	var errs []error

	// Create and add nodes for each run
	nodes, err := t.createNodes(g)
	if err != nil {
		return err
	}

	// Connect nodes based on dependencies
	if err := t.connectDependencies(g, nodes); err != nil {
		errs = append(errs, err)
	}

	// Connect nodes with the same state key sequentially
	if err := t.connectStateKeyRuns(g, nodes); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (t *TestRunTransformer) createNodes(g *terraform.Graph) ([]*NodeTestRun, error) {
	var nodes []*NodeTestRun
	var prev *NodeTestRun
	for _, run := range t.File.Runs {
		node := &NodeTestRun{run: run, file: t.File}
		g.Add(node)
		nodes = append(nodes, node)

		if prev != nil {
			parallelized := prev.run.Config.Parallel && run.Config.Parallel
			// we connect 2 sequential runs IF
			// 1. at least one of them is NOT eligible for parallelization OR
			// 2. they are both eligible for parallelization AND have the same state key
			if !parallelized || (parallelized && prev.run.GetStateKey() == run.GetStateKey()) {
				g.Connect(dag.BasicEdge(node, prev))
			}
		}
		prev = node

	}
	return nodes, nil
}

func (t *TestRunTransformer) connectDependencies(g *terraform.Graph, nodes []*NodeTestRun) error {
	var errs []error
	nodeMap := make(map[string]*NodeTestRun)
	for _, node := range nodes {
		nodeMap[node.run.Name] = node
	}
	for _, node := range nodes {
		refs, err := getRefs(node.run)
		if err != nil {
			return err
		}
		for _, ref := range refs {
			subjectStr := ref.Subject.String()
			if !strings.HasPrefix(subjectStr, "run.") {
				continue
			}
			runName := strings.TrimPrefix(subjectStr, "run.")
			if runName == "" {
				continue
			}
			dependency, ok := nodeMap[runName]
			if !ok {
				errs = append(errs, fmt.Errorf("dependency `run.%s` not found for run %q", runName, node.run.Name))
				continue
			}
			g.Connect(dag.BasicEdge(node, dependency))
		}
	}
	return errors.Join(errs...)
}

func (t *TestRunTransformer) connectStateKeyRuns(g *terraform.Graph, nodes []*NodeTestRun) error {
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
	return nil
}

func getRefs(run *moduletest.Run) ([]*addrs.Reference, error) {
	refs, refDiags := run.GetReferences()
	if refDiags.HasErrors() {
		return nil, refDiags.Err()
	}
	for _, expr := range run.Config.Variables {
		moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		if moreDiags.HasErrors() {
			return nil, moreDiags.Err()
		}
		refs = append(refs, moreRefs...)
	}
	return refs, nil
}

// -------------------------------------------------------- CloseTestGraphTransformer --------------------------------------------------------

// CloseTestGraphTransformer is a GraphTransformer that adds a root to the graph.
type CloseTestGraphTransformer struct{}

func (t *CloseTestGraphTransformer) Transform(g *terraform.Graph) error {
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

func (n *nodeCloseTest) Name() string {
	return "testroot"
}
