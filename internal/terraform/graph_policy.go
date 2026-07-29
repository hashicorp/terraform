// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock      sync.Mutex
	resources addrs.Map[addrs.AbsResourceInstance, *nodeResourcePolicy]
	queries   addrs.Map[addrs.AbsResourceInstance, *nodeQueryResourcePolicy]

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span
}

func newPolicySubgraph() *policySubgraph {
	return &policySubgraph{
		resources: addrs.MakeMap[addrs.AbsResourceInstance, *nodeResourcePolicy](),
		queries:   addrs.MakeMap[addrs.AbsResourceInstance, *nodeQueryResourcePolicy](),
	}
}

func (ps *policySubgraph) Add(node *nodeResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	existing, ok := ps.resources.GetOk(node.ResourceAddr)
	if !ok {
		ps.resources.Put(node.ResourceAddr, node)
		return
	}

	// a node already exists. If there is a create/delete combination,
	// then this originated from a replacement operation (replace or CBD),
	// and we will create a new replace node to represent it, so that
	// we have one single policy node to represent the replacement operation.
	var before, after cty.Value
	if existing.Action == plans.Create && node.Action == plans.Delete {
		before = node.Before
		after = existing.After
	} else if existing.Action == plans.Delete && node.Action == plans.Delete {
		before = existing.Before
		after = node.After
	}
	replaceNode := &nodeResourcePolicy{
		ResourceAddr: node.ResourceAddr,
		ProviderAddr: node.ProviderAddr,
		Before:       before,
		After:        after,
		Action:       plans.CreateThenDelete,
	}
	ps.resources.Put(node.ResourceAddr, replaceNode)
}

func (ps *policySubgraph) AddQuery(node *nodeQueryResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.queries.Put(node.ResourceAddr, node)
}

func (ps *policySubgraph) evalGraph(span trace.Span) *Graph {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.span = span

	g := &Graph{}
	// create a finish node that depends on all policy nodes
	finish := &nodePolicyEvalFinish{span: span}
	g.Add(finish)
	// add resource policy nodes to the graph
	for _, node := range ps.resources.Iter() {
		g.Add(node)

		// Wire finish to policy node types
		g.Connect(finish, node)
	}
	for _, node := range ps.queries.Iter() {
		g.Add(node)
		g.Connect(finish, node)
	}

	// ensure the graph has a single root
	addRootNodeToGraph(g)
	return g
}
