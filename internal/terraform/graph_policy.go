// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock  sync.Mutex
	graph Graph

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span

	// run describes the policy run of the walk, or is nil when the walk
	// has none.
	run *policyRunOpts

	// configVals are the unmarked configuration values of the managed
	// resource instances planned during the walk, by address, so origins can
	// check that the planned value keeps the configured one. It's non-nil
	// only when the walk is a plan that relates instances.
	configVals map[string]cty.Value

	// relations are the relations of the walk's instances. They are set in
	// nodePolicyEval.DynamicExpand before the subject nodes run, when the
	// walk relates instances.
	relations *policyRelations

	// generation is the policy phase of the walk's objects, which
	// nodePolicyEval.DynamicExpand starts.
	generation uint64
}

func newPolicySubgraph() *policySubgraph {
	return newPolicySubgraphForRun(nil)
}

// newPolicySubgraphForRun returns a policy subgraph for a walk with the given
// relationships, which may be nil.
func newPolicySubgraphForRun(run *policyRunOpts) *policySubgraph {
	ps := &policySubgraph{run: run}
	if run.relating() && !run.Apply {
		ps.configVals = make(map[string]cty.Value)
	}
	return ps
}

// resourceConfigured records the configuration value that a managed resource
// instance was planned with. It has no effect unless the walk is a plan with
// relationships.
func (ps *policySubgraph) resourceConfigured(addr addrs.AbsResourceInstance, val cty.Value) {
	if ps.configVals == nil {
		return
	}
	val, _ = val.UnmarkDeep()
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.configVals[addr.String()] = val
}

// resourceConfigValue returns the configuration value that a managed
// resource instance was planned with during the walk; ok is false if it
// wasn't recorded.
func (ps *policySubgraph) resourceConfigValue(addr addrs.AbsResourceInstance) (val cty.Value, ok bool) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	val, ok = ps.configVals[addr.String()]
	return val, ok
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

// walkRelations returns the relations of the walk's instances, or nil if the
// walk relates none.
func (ps *policySubgraph) walkRelations() *policyRelations {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.relations
}

// appliedChange returns the applied plan's change of the given object of a
// resource instance, or nil if there is none or the walk isn't an apply.
func (ps *policySubgraph) appliedChange(addr addrs.AbsResourceInstance, key states.DeposedKey) *plans.ResourceInstanceChange {
	if ps.run == nil {
		return nil
	}
	for _, change := range ps.run.AppliedChanges {
		if change.DeposedKey == key && change.Addr.Equal(addr) {
			return change
		}
	}
	return nil
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
