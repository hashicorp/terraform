// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
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

	// Collect all managed resource instance nodes and all provider closer
	// nodes that are already in the graph.
	var resourceNodes []dag.Vertex
	var closeProviderNodes []dag.Vertex

	for v := range g.VerticesSeq() {
		if ri, ok := v.(GraphNodeConfigResource); ok {
			addr := ri.ResourceAddr()
			if addr.Resource.Mode == addrs.ManagedResourceMode {
				resourceNodes = append(resourceNodes, v)
			}
		}

		if _, ok := v.(GraphNodeCloseProvider); ok {
			closeProviderNodes = append(closeProviderNodes, v)
		}
	}

	// If there are no managed resources at all, there is nothing to evaluate
	// policy against.
	if len(resourceNodes) == 0 {
		return nil
	}

	policyNode := &nodePolicyEval{}
	g.Add(policyNode)

	// The policy node must execute after every managed resource instance node.
	for _, rsNode := range resourceNodes {
		g.Connect(dag.BasicEdge(policyNode, rsNode))
	}

	// We keep the provider open until after policy evaluation so that the
	// policy engine callbacks can still use them. For example, policies may need to access data
	// sources of a provider.
	for _, providerNode := range closeProviderNodes {
		g.Connect(dag.BasicEdge(providerNode, policyNode))
	}

	return nil
}
