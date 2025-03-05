// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"sync"
)

// Filter provides a concurrent-friendly filtering mechanism
type Filter struct {
	mu       *sync.RWMutex
	includes Set
	excludes Set
}

// AllowedStatus represents the detailed status of an item in the filter
type AllowedStatus int

const (
	None AllowedStatus = iota
	Allowed
	ExplicitlyExcluded
)

// NewFilter creates a new Filter
func NewFilter() *Filter {
	return &Filter{
		mu:       &sync.RWMutex{},
		includes: make(Set),
		excludes: make(Set),
	}
}

// Include adds items to the include list
func (f *Filter) Include(items ...Vertex) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, item := range items {
		f.includes[item] = struct{}{}
	}
}

// Exclude adds items to the exclude list
func (f *Filter) Exclude(items ...Vertex) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, item := range items {
		f.excludes[item] = struct{}{}
	}
}

// Allowed checks if an item is allowed based on the include and exclude lists,
// with the exclude list taking precedence
func (f *Filter) Allowed(item Vertex) bool {
	return f.status(item) == Allowed || f.status(item) == None
}

func (f *Filter) status(item Vertex) AllowedStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// If in excludes, it's not allowed
	if _, excluded := f.excludes[item]; excluded {
		return ExplicitlyExcluded
	}

	// Check if item is in includes
	_, included := f.includes[item]
	if !included {
		return None
	}
	return Allowed
}

// Matches checks if the item matches the given status in the filter
func (f *Filter) Matches(item Vertex, status AllowedStatus) bool {
	return f.status(item) == status
}
