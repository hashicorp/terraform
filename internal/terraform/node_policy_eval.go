// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/trace"
)

// nodePolicyEval is a node that completes the building of the policy graph,
// with incoming edges from the resource graph so that policy evaluation
// is performed only when the resource graph is complete.
type nodePolicyEval struct {
	// a dependency map of resource nodes. This also includes transitive dependencies.
	resourceDepMap collections.Map[dag.Vertex, dag.VertexSet]
}

var _ GraphNodeDynamicExpandable = (*nodePolicyEval)(nil)
var _ dag.ErroredDependencyHandler = (*nodePolicyEval)(nil)

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

// OnErroredDependencies allows failures from upstream nodes to be tolerated
// so that the policy evaluation can proceed even if some resource instance nodes
// evaluated with error diagnostics.
func (n *nodePolicyEval) OnErroredDependencies(deps ...dag.Vertex) dag.DependencyResult {
	dependencyResult := dag.DependencyResultSoftFailure
	// Loop through all the dependencies in the graph
	for _, dep := range deps {
		// If a dependency failed and is not a resource instance, then
		// return a hard failure early
		if _, ok := dep.(GraphNodeConfigResource); !ok {
			return dag.DependencyResultHardFailure
		}

		// Get transitive dependencies as well.
		depDeps, ok := n.resourceDepMap.GetOk(dep)
		if !ok {
			continue
		}

		for range depDeps.All() {
			// If a dependency failed and is not a resource instance, then
			// return a hard failure early
			if _, ok := dep.(GraphNodeConfigResource); !ok {
				return dag.DependencyResultHardFailure
			}

			// this is a node resource, so we are probably returning a soft failure
		}
	}

	return dependencyResult
}

// nodePolicyEvalFinish is a sentinel node appended to the policy subgraph that
// runs after every policy node and ends the policy-execution phase span. It
// must tolerate upstream failures so the span is still closed even if a policy
// node returned error diagnostics.
type nodePolicyEvalFinish struct {
	span trace.Span
}

var _ GraphNodeExecutable = (*nodePolicyEvalFinish)(nil)
var _ dag.ErroredDependencyHandler = (*nodePolicyEvalFinish)(nil)

func (n *nodePolicyEvalFinish) Name() string {
	return "(policy evaluation complete)"
}

func (n *nodePolicyEvalFinish) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	n.span.End()
	return nil
}

// OnErroredDependencies returns a soft failure so that this node still executes
// even if dependencies errored
func (n *nodePolicyEvalFinish) OnErroredDependencies(deps ...dag.Vertex) dag.DependencyResult {
	return dag.DependencyResultSoftFailure
}
