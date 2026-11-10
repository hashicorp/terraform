// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/zclconf/go-cty/cty"
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

	// run describes the relationships of this walk, or is nil when the walk
	// has none.
	run *policyRunOpts

	// providerConfigs are the values that the walk configured providers
	// with, by provider configuration address. It's non-nil only when the
	// walk has relationships.
	providerConfigs map[string]cty.Value

	// configVals are the unmarked configuration values of the managed
	// resource instances planned during the walk, by address, so origins can
	// check that the planned value keeps the configured one. It's non-nil
	// only when the walk is a plan with relationships.
	configVals map[string]cty.Value

	// relations are the relations of the walk's instances. They are set in
	// nodePolicyEval.DynamicExpand before the subject nodes run, when the
	// walk has relationships and no shared run.
	relations *PolicyRelations
}

func newPolicySubgraph() *policySubgraph {
	return newPolicySubgraphForRun(nil)
}

// newPolicySubgraphForRun returns a policy subgraph for a walk with the given
// relationships, which may be nil.
func newPolicySubgraphForRun(run *policyRunOpts) *policySubgraph {
	ps := &policySubgraph{run: run}
	if run != nil {
		ps.providerConfigs = make(map[string]cty.Value)
		if !run.Apply {
			ps.configVals = make(map[string]cty.Value)
		}
	}
	return ps
}

// providerConfigured records the value that a provider was configured with.
// It has no effect unless the walk has relationships.
func (ps *policySubgraph) providerConfigured(addr addrs.AbsProviderConfig, cfg cty.Value) {
	if ps.providerConfigs == nil {
		return
	}
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.providerConfigs[addr.String()] = cfg
}

// providerConfig returns the value that the walk configured the provider at
// the given address with; ok is false if it didn't configure it.
func (ps *policySubgraph) providerConfig(addr string) (cfg cty.Value, ok bool) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	cfg, ok = ps.providerConfigs[addr]
	return cfg, ok
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
// walk has no relationships or a shared run.
func (ps *policySubgraph) walkRelations() *PolicyRelations {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.relations
}

// subjectDeposedKey returns the deposed key that the evaluation of the given
// object of a resource instance sends: the object's deposed key if the
// walk has relationships and a planned change for that object, and ""
// otherwise. So the deposed object of a create-before-destroy replace, whose
// key the apply allocates, has the key of the replace's change, "".
func (ps *policySubgraph) subjectDeposedKey(addr addrs.AbsResourceInstance, key states.DeposedKey) string {
	if key == states.NotDeposed || ps.run == nil {
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
