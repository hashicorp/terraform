// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock  sync.Mutex
	graph Graph

	// spanCtx and span carry the "terraform.policy.evaluate" phase span that
	// brackets the policy subgraph walk. They are set by nodePolicyEval before
	// the subgraph is walked and read by the per-resource policy nodes so that
	// their evaluation spans nest under the single policy-execution phase span
	// (which is itself a child of the enclosing plan/apply span). The span is
	// ended by nodePolicyEvalFinish once every policy node has run.
	spanCtx context.Context
	span    trace.Span
}

func newPolicySubgraph() *policySubgraph {
	var g Graph
	return &policySubgraph{graph: g}
}

func (ps *policySubgraph) Add(node *nodeResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.graph.Add(node)
}

// setPhaseSpan records the policy-execution phase span and its context. It is
// called once, by nodePolicyEval, before the subgraph walk begins.
func (ps *policySubgraph) setPhaseSpan(ctx context.Context, span trace.Span) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.spanCtx = ctx
	ps.span = span
}

// phaseSpanContext returns the context carrying the policy-execution phase
// span, or nil if no phase span was set (e.g. in unit tests that drive the
// nodes directly). Callers should fall back to their own context when this
// returns nil.
func (ps *policySubgraph) phaseSpanContext() context.Context {
	if ps == nil {
		return nil
	}
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.spanCtx
}

// endPhaseSpan ends the policy-execution phase span if one was set. It is safe
// to call multiple times.
func (ps *policySubgraph) endPhaseSpan() {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	if ps.span != nil {
		ps.span.End()
		ps.span = nil
	}
}
