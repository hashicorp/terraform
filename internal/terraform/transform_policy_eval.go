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

	// Collect all provider closer nodes and all of the terminal nodes in the
	// main graph, ignoring provider closer nodes. The policy node must wait for
	// those terminal nodes so that all mutations to changes and state are
	// complete before policy evaluation closes those objects for read-only
	// callbacks.
	var hasManagedResourceNode bool
	var terminalNodes []dag.Vertex
	var closeProviderNodes []dag.Vertex

	for v := range g.VerticesSeq() {
		if _, ok := v.(GraphNodeConfigResource); ok {
			terminalNodes = append(terminalNodes, v)
			hasManagedResourceNode = true
			continue
		}

		// select close provider nodes
		if _, ok := v.(GraphNodeCloseProvider); ok {
			closeProviderNodes = append(closeProviderNodes, v)
			continue
		}

		// select terminal nodes
		if g.UpEdges(v).Len() == 0 {
			terminalNodes = append(terminalNodes, v)
			continue
		}
	}

	// If there are no managed resources at all, there is nothing to evaluate
	// policy against.
	if !hasManagedResourceNode {
		return nil
	}

	policyNode := &nodePolicyEval{}
	g.Add(policyNode)

	// Connect the policy node to every terminal node so that it
	// executes only after all remaining graph work that can still mutate state
	// or changes has completed.
	for _, node := range terminalNodes {
		g.Connect(dag.BasicEdge(policyNode, node))
	}

	// We keep the provider open until after policy evaluation so that the
	// policy engine callbacks can still use them. For example, policies may need to access data
	// sources of a provider.
	for _, providerNode := range closeProviderNodes {
		g.Connect(dag.BasicEdge(providerNode, policyNode))
	}

	return nil
}
