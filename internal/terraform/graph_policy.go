// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock  sync.Mutex
	graph Graph

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span

	// runID is the id of the relationship run of this walk, or "" when there
	// is none. It is set in nodePolicyEval.DynamicExpand before the subject
	// nodes run.
	runID string

	// run describes the relationship run of this walk, or is nil when the
	// walk has none.
	run *policyRunOpts

	// providers records the provider configurations configured during the
	// walk. It's non-nil only when the walk has a relationship run.
	providers *policyProviderTable
}

func newPolicySubgraph() *policySubgraph {
	return newPolicySubgraphForRun(nil)
}

// newPolicySubgraphForRun returns a policy subgraph for a walk with the given
// relationship run, which may be nil.
func newPolicySubgraphForRun(run *policyRunOpts) *policySubgraph {
	ps := &policySubgraph{run: run}
	if run != nil {
		ps.providers = newPolicyProviderTable()
	}
	return ps
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

// RunID returns the id of the relationship run of this walk, or "" when there
// is no run.
func (ps *policySubgraph) RunID() string {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.runID
}

// subjectDeposedKey returns the deposed key that the evaluation of the given
// object of a resource instance sends: the object's deposed key if the
// walk's relationship run has a planned change for that object, and ""
// otherwise. So the deposed object of a create-before-destroy replace, whose
// key the apply allocates, has the key of the replace's change, "".
func (ps *policySubgraph) subjectDeposedKey(addr addrs.AbsResourceInstance, key states.DeposedKey) string {
	if key == states.NotDeposed || ps.run == nil || ps.RunID() == "" {
		return ""
	}
	for _, change := range ps.run.AppliedChanges {
		if change.DeposedKey == key && change.Addr.Equal(addr) {
			return key.String()
		}
	}
	return ""
}

func (ps *policySubgraph) evalGraph(span trace.Span) *Graph {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.span = span

	finish := &nodePolicyEvalFinish{span: span}
	ps.graph.Add(finish)
	for pn := range ps.graph.VerticesSeq() {
		// Wire finish only to policy node types; all other vertices are skipped.
		switch pn.(type) {
		case *nodeResourcePolicy, *nodeQueryResourcePolicy:
		default:
			continue
		}
		// finish depends on pn, so pn runs first and finish runs after.
		ps.graph.Connect(finish, pn)
	}

	return &ps.graph
}
