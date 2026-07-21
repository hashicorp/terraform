// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/dag"
	"go.opentelemetry.io/otel/trace"
)

// policySubgraph is a subgraph that stores resource policy nodes.
type policySubgraph struct {
	lock  sync.Mutex
	graph Graph

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span
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
	finish := &nodePolicyEvalFinish{span: span}
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
