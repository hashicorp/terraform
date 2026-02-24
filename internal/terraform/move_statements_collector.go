// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"sync"

	"github.com/hashicorp/terraform/internal/refactoring"
)

type moveStatementsCollector struct {
	mu    sync.Mutex
	items []collectedMoveStatement
}

type collectedMoveStatement struct {
	stmtIndex    int
	expandIndex  int
	statementVal refactoring.MoveStatement
}

func newMoveStatementsCollector() *moveStatementsCollector {
	return &moveStatementsCollector{}
}

func (c *moveStatementsCollector) Record(stmtIndex, expandIndex int, stmt refactoring.MoveStatement) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = append(c.items, collectedMoveStatement{
		stmtIndex:    stmtIndex,
		expandIndex:  expandIndex,
		statementVal: stmt,
	})
}

func (c *moveStatementsCollector) Results() []refactoring.MoveStatement {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]collectedMoveStatement, len(c.items))
	copy(items, c.items)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].stmtIndex != items[j].stmtIndex {
			return items[i].stmtIndex < items[j].stmtIndex
		}
		return items[i].expandIndex < items[j].expandIndex
	})

	ret := make([]refactoring.MoveStatement, len(items))
	for i, item := range items {
		ret[i] = item.statementVal
	}
	return ret
}
