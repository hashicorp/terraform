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

	// Open the policy-execution phase span. This brackets the entire policy
	// subgraph walk, so it cleanly separates the policy phase (which runs
	// after all resources have been planned/applied) from the main resource
	// graph work in the trace. The per-resource policy evaluation spans are
	// parented under this span (see evaluatePolicies), and it is ended by the
	// finish node added below once every policy node has executed.
	//
	// The parent is ctx.StopCtx() -- the run context, which is itself a child
	// of the enclosing "terraform plan"/"terraform apply" span -- so the
	// phase span nests directly under the command/operation span.
	spanCtx, span := tracer().Start(ctx.StopCtx(), "terraform.policy.evaluate")
	policyGraph.setPhaseSpan(spanCtx, span)

	// Add a finish node that depends on every policy node, so it runs last and
	// ends the phase span once all policy evaluation has completed. We collect
	// the policy nodes before adding the finish node so the finish node does
	// not depend on itself.
	finish := &nodePolicyEvalFinish{policyGraph: policyGraph}
	var policyNodes []dag.Vertex
	for v := range policyGraph.graph.VerticesSeq() {
		policyNodes = append(policyNodes, v)
	}
	policyGraph.graph.Add(finish)
	for _, pn := range policyNodes {
		// finish depends on pn, so pn runs first and finish runs after.
		policyGraph.graph.Connect(dag.BasicEdge(finish, pn))
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

// nodePolicyEvalFinish is a sentinel node appended to the policy subgraph that
// runs after every policy node and ends the policy-execution phase span. It
// must tolerate upstream failures so the span is still closed even if a policy
// node returned error diagnostics.
type nodePolicyEvalFinish struct {
	policyGraph *policySubgraph
}

var _ GraphNodeExecutable = (*nodePolicyEvalFinish)(nil)
var _ dag.TolerantVertex = (*nodePolicyEvalFinish)(nil)

func (n *nodePolicyEvalFinish) Name() string {
	return "(policy evaluation complete)"
}

func (n *nodePolicyEvalFinish) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	n.policyGraph.endPhaseSpan()
	return nil
}

// AllowUpstreamFailure tolerates failures from the policy nodes so the phase
// span is always ended.
func (n *nodePolicyEvalFinish) AllowUpstreamFailure(dep dag.Vertex) bool {
	_, ok := dep.(*nodeResourcePolicy)
	return ok
}
