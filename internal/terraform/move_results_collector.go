// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/refactoring"
)

// moveResultsCollector accumulates move results safely across parallel graph
// node execution.
type moveResultsCollector struct {
	mu      sync.Mutex
	results refactoring.MoveResults
}

func newMoveResultsCollector() *moveResultsCollector {
	return &moveResultsCollector{
		results: refactoring.MakeMoveResults(),
	}
}

func (c *moveResultsCollector) RecordOldAddr(oldAddr, newAddr addrs.AbsResourceInstance) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if prevMove, exists := c.results.Changes.GetOk(oldAddr); exists {
		// Collapse chains (A->B, then B->C) into a single A->C result entry.
		c.results.Changes.Remove(oldAddr)
		oldAddr = prevMove.From
	}
	c.results.Changes.Put(newAddr, refactoring.MoveSuccess{
		From: oldAddr,
		To:   newAddr,
	})
}

func (c *moveResultsCollector) RecordBlockage(actualAddr, wantedAddr addrs.AbsMoveable) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.results.Blocked.Put(actualAddr, refactoring.MoveBlocked{
		Wanted: wantedAddr,
		Actual: actualAddr,
	})
}

func (c *moveResultsCollector) Results() refactoring.MoveResults {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.results
}
