// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/dag"
)

// graphFilter manages inclusion and exclusion of graph nodes during a walk.
type graphFilter struct {
	mu       *sync.RWMutex
	includes dag.Set
	excludes dag.Set
}

type Status int

const (
	None Status = iota
	NodeIncluded
	NodeExcluded
)

// newFilter creates a new GraphFilter
func newFilter() *graphFilter {
	return &graphFilter{
		mu:       &sync.RWMutex{},
		includes: make(dag.Set),
		excludes: make(dag.Set),
	}
}

// Include adds items to the include list
func (f *graphFilter) Include(items ...dag.Vertex) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, item := range items {
		f.includes[item] = struct{}{}
	}
}

// Exclude adds items to the exclude list
func (f *graphFilter) Exclude(items ...dag.Vertex) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, item := range items {
		f.excludes[item] = struct{}{}
	}
}

// NodeAllowed checks whether a node is allowed in traversal.
// A node is allowed if it's included or not explicitly excluded.
func (f *graphFilter) NodeAllowed(item dag.Vertex) bool {
	return f.status(item) != NodeExcluded
}

func (f *graphFilter) status(item dag.Vertex) Status {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// If in excludes, it's not allowed
	if _, excluded := f.excludes[item]; excluded {
		return NodeExcluded
	}

	// Check if item is in includes
	_, included := f.includes[item]
	if !included {
		return None
	}
	return NodeIncluded
}

// Matches checks if the item matches the specified status
func (f *graphFilter) Matches(item dag.Vertex, status Status) bool {
	return f.status(item) == status
}
