// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/trace"
)

// nodePolicyEval is a node that completes the building of the policy graph,
// with incoming edges from the resource graph so that policy evaluation
// is performed only when the resource graph is complete.
type nodePolicyEval struct {
}

var _ GraphNodeDynamicExpandable = (*nodePolicyEval)(nil)
var _ dag.AlwaysRunVertex = (*nodePolicyEval)(nil)

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

	spanCtx, span := tracer().Start(ctx.StopCtx(), "terraform.policy.evaluate")
	var runDiags policy.Diagnostics
	if policyGraph.run != nil {
		runDiags = policyGraph.startRelationshipRun(ctx, spanCtx)
	}
	var diags tfdiags.Diagnostics
	if len(runDiags) > 0 {
		// Like policy setup diagnostics, the diagnostics of beginning the
		// relationship run go to the view and don't fail the walk.
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PolicyDiagnostics(runDiags)
		}))
	}
	if runDiags.HasErrors() {
		// The relationship run didn't begin, so no policy is evaluated. There
		// is no subgraph whose finish node ends the span, so end it here.
		span.End()
		return nil, diags
	}
	return policyGraph.evalGraph(span), diags
}

// AlwaysRun implements [dag.AlwaysRunVertex] so that the policy evaluation
// can proceed even if some resource instance nodes evaluated with error diagnostics.
func (n *nodePolicyEval) AlwaysRun() {}

// nodePolicyEvalFinish is a sentinel node appended to the policy subgraph that
// runs after every policy node and ends the policy-execution phase span. It
// must tolerate upstream failures so the span is still closed even if a policy
// node returned error diagnostics.
type nodePolicyEvalFinish struct {
	span trace.Span
}

var _ GraphNodeExecutable = (*nodePolicyEvalFinish)(nil)
var _ dag.AlwaysRunVertex = (*nodePolicyEvalFinish)(nil)

func (n *nodePolicyEvalFinish) Name() string {
	return "(policy evaluation complete)"
}

func (n *nodePolicyEvalFinish) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	if pg := ctx.PolicyGraph(); pg != nil {
		pg.finishRelationshipRun(ctx, trace.ContextWithSpan(ctx.StopCtx(), n.span))
	}
	n.span.End()
	return nil
}

// AlwaysRun implements [dag.AlwaysRunVertex] so that this node still executes
// even if dependencies errored
func (n *nodePolicyEvalFinish) AlwaysRun() {}
