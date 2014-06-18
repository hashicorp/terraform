package diff

import (
	"sync"
)

// LazyResourceMap is a way to lazy-load resource builders.
//
// By lazy loading resource builders, a considerable amount of compute
// effort for building the builders can be avoided. This is especially
// helpful in Terraform providers that support many resource types.
type LazyResourceMap struct {
	Resources map[string]ResourceBuilderFactory

	l        sync.Mutex
	memoized map[string]*ResourceBuilder
}

// ResourceBuilderFactory is a factory function for creating a resource
// builder that is used for lazy loading resource builders in the Builder
// struct.
type ResourceBuilderFactory func() *ResourceBuilder

// Get gets the ResourceBuilder for the given resource type, and returns
// nil if the resource builder cannot be found.
//
// This will memoize the result, returning the same result for the same
// type if called again.
func (m *LazyResourceMap) Get(r string) *ResourceBuilder {
	m.l.Lock()
	defer m.l.Unlock()

	// If we have it saved, return that
	if rb, ok := m.memoized[r]; ok {
		return rb
	}

	// Get the factory function
	f, ok := m.Resources[r]
	if !ok {
		return nil
	}

	// Save it so that we don't rebuild
	if m.memoized == nil {
		m.memoized = make(map[string]*ResourceBuilder)
	}
	m.memoized[r] = f()

	return m.memoized[r]
}
