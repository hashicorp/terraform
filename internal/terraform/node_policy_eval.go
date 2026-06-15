// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodePolicyEval is a node that completes the building of the policy graph,
// with incoming edges from the resource graph so that policy evaluation
// is performed only when the resource graph is complete.
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
	// Close the changes sync to prevent writes during policy evaluation
	ctx.Changes().Close()
	ctx.State().Close()

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
