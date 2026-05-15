// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"
)

type policySubgraph struct {
	lock  sync.Mutex
	graph Graph
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
