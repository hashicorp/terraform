// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"

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

// newPolicySemaphore creates a Semaphore for policy evaluation with a default
// capacity of GOMAXPROCS. TF_POLICY_PARALLELISM env variable override for debugging.
func newPolicySemaphore() Semaphore {
	n := runtime.GOMAXPROCS(0)
	if v := os.Getenv("TF_POLICY_PARALLELISM"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		} else {
			log.Printf("[WARN] TF_POLICY_PARALLELISM %q is not a valid positive integer, defaulting to GOMAXPROCS (%d)", v, n)
		}
	}
	return NewSemaphore(n)
}
