// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
	"go.opentelemetry.io/otel/trace"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock  sync.Mutex
	graph Graph

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span

	// timeouts holds the per-call and overall deadline durations for policy
	// evaluation. These are set once before the graph is walked and read
	// concurrently during the walk; the lock is NOT held while reading them
	// because they are written only in DynamicExpand, which happens before
	// any Execute calls begin.
	timeouts policy.EvalTimeouts

	// deadlineCtx is derived from the incoming request context and expires
	// when the overall policy-pass deadline is reached. It is stored via an
	// atomic.Pointer so concurrent Execute goroutines can read it without
	// holding the lock. A nil stored value means no overall deadline is active.
	// deadlineCancel is protected by lock and accessed only from setDeadline
	// and cancelDeadline, which are not called concurrently.
	deadlineCtx    atomic.Pointer[context.Context]
	deadlineCancel context.CancelFunc
}

func newPolicySubgraph() *policySubgraph {
	var g Graph
	return &policySubgraph{
		graph:    g,
		timeouts: policy.DefaultEvalTimeouts(),
	}
}

// setDeadline installs an overall-pass deadline on the subgraph. It is called
// once from nodePolicyEval.DynamicExpand before the graph walk begins.
// parent must already be a context that is cancelled when the outer request
// is cancelled, so the deadline never outlives the parent context.
func (ps *policySubgraph) setDeadline(parent context.Context) {
	if ps.timeouts.Overall <= 0 {
		// Overall deadline is disabled; nothing to install.
		return
	}

	ctx, cancel := context.WithTimeout(parent, ps.timeouts.Overall)
	ps.deadlineCtx.Store(&ctx)

	ps.lock.Lock()
	ps.deadlineCancel = cancel
	ps.lock.Unlock()
}

// cancelDeadline cancels the deadline context installed by setDeadline. It is
// safe to call even if setDeadline was never called or if Overall is 0.
func (ps *policySubgraph) cancelDeadline() {
	ps.lock.Lock()
	cancel := ps.deadlineCancel
	ps.lock.Unlock()

	if cancel != nil {
		cancel()
	}
}

// overallDeadlineExceeded returns true if the overall pass deadline has fired.
// It is safe to call concurrently from multiple Execute goroutines; the context
// pointer is loaded via an atomic operation.
func (ps *policySubgraph) overallDeadlineExceeded() bool {
	ptr := ps.deadlineCtx.Load()
	if ptr == nil {
		return false
	}
	select {
	case <-(*ptr).Done():
		return true
	default:
		return false
	}
}

func (ps *policySubgraph) Add(node *nodeResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.graph.Add(node)
}

func (ps *policySubgraph) AddQuery(node *nodeQueryResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.graph.Add(node)
}

func (ps *policySubgraph) evalGraph(span trace.Span) *Graph {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.span = span

	g := ps.graphCopyLocked()
	finish := &nodePolicyEvalFinish{span: span, policyGraph: ps}
	g.Add(finish)
	for pn := range g.VerticesSeq() {
		// Wire finish only to policy node types; all other vertices are skipped.
		switch pn.(type) {
		case *nodeResourcePolicy, *nodeQueryResourcePolicy:
		default:
			continue
		}
		// finish depends on pn, so pn runs first and finish runs after.
		g.Connect(dag.BasicEdge(finish, pn))
	}

	// ensure the graph has a single root
	addRootNodeToGraph(&g)
	return &g
}

func (ps *policySubgraph) graphCopyLocked() Graph {
	var g Graph
	for _, v := range ps.graph.Vertices() {
		g.Add(v)
	}
	for _, e := range ps.graph.Edges() {
		g.Connect(e)
	}
	return g
}
