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
	var prev *NodeTestRun
	var errs []error
	runsSoFar := make(map[string]*NodeTestRun)
	for _, run := range t.File.Runs {
		// If we're testing a specific configuration, we need to use that
		config := t.config
		if run.Config.ConfigUnderTest != nil {
			config = run.Config.ConfigUnderTest
		}

		node := &NodeTestRun{run: run, file: t.File, config: config}
		g.Add(node)
		if prev != nil {
			g.Connect(dag.BasicEdge(node, prev))
		}
		prev = node

		// Connect the run to all the other runs that it depends on
		refs, err := getRefs(run)
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
			dependency, ok := runsSoFar[runName]
			if !ok {
				errs = append(errs, fmt.Errorf("dependency `run.%s` not found for run %q", runName, run.Name))
				continue
			}
			g.Connect(dag.BasicEdge(node, dependency))
		}
		runsSoFar[run.Name] = node
	}

	return errors.Join(errs...)
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
