// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodePolicyEval is a node that evaluates policy for all resource instances
// after they have been planned or applied,
// so that the complete resource graph state is available to the policy engine.
type nodePolicyEval struct{}

var _ GraphNodeDynamicExpandable = (*nodePolicyEval)(nil)
var _ dag.TolerantVertex = (*nodePolicyEval)(nil)

func (n *nodePolicyEval) Name() string {
	return "(evaluate policies)"
}

func (n *nodePolicyEval) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	policyGraph := ctx.PolicyGraph()
	if policyGraph == nil {
		log.Printf("[DEBUG] policyGraph is nil")
		return nil, nil
	}
	// ensure the graph has a single root
	addRootNodeToGraph(&policyGraph.graph)
	return &policyGraph.graph, nil
}

// AllowUpstreamFailure allows failures from upstream nodes to be tolerated
// so that the policy evaluation can proceed even if some resource instance nodes
// evaluated with error diagnostics.
func (n *nodePolicyEval) AllowUpstreamFailure(dep dag.Vertex) bool {
	_, ok := dep.(GraphNodeConfigResource)
	return ok
}
