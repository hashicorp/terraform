// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
)

// policyEvalTransformer is a transformer that adds the policy evaluation
// node to the graph. it also wires up the dependency edges to ensure that the
// node is executed after all resources have been planned or applied, and that
// providers are kept open until all policies have been evaluated.
type policyEvalTransformer struct {
	PolicyClient policy.Client
}

var _ GraphTransformer = (*policyEvalTransformer)(nil)

func (t *policyEvalTransformer) Transform(g *Graph) error {
	if t.PolicyClient == nil {
		return nil
	}
	policyNode := &nodePolicyEval{}
	g.Add(policyNode)

	for v := range g.VerticesSeq() {
		if v == policyNode {
			continue
		}
		// Connect provider closer nodes to policy so that
		// we keep all providers open until after policy evaluation so that the
		// policy engine callbacks can still use them. For example, policies may need to access data
		// sources of a provider.
		if _, ok := v.(GraphNodeCloseProvider); ok {
			g.Connect(dag.BasicEdge(v, policyNode))
			continue
		}

		// Connect the policy node to every non-provider closer node so that it
		// executes only after all remaining graph work that can still mutate state
		// or changes has completed.
		g.Connect(dag.BasicEdge(policyNode, v))
	}

	return nil
}
