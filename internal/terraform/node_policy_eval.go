// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/trace"
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
	// Close the changes/state objects to prevent writes during policy evaluation.
	// This is safe to do because policy evaluation is the final step in the plan/apply process.
	// If any future nodes attempt to write to these states, they will panic.
	ctx.Changes().Close()
	ctx.State().Close()

	_, span := tracer().Start(ctx.StopCtx(), "terraform.policy.evaluate")
	return policyGraph.evalGraph(span), nil
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
	span trace.Span
}

var _ GraphNodeExecutable = (*nodePolicyEvalFinish)(nil)
var _ dag.TolerantVertex = (*nodePolicyEvalFinish)(nil)

func (n *nodePolicyEvalFinish) Name() string {
	return "(policy evaluation complete)"
}

func (n *nodePolicyEvalFinish) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	n.span.End()
	return nil
}

// AllowUpstreamFailure tolerates failures from the policy nodes so the phase
// span is always ended.
func (n *nodePolicyEvalFinish) AllowUpstreamFailure(dep dag.Vertex) bool {
	switch dep.(type) {
	case *nodeResourcePolicy, *nodeQueryResourcePolicy:
		return true
	}
	return false
}
