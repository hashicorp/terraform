// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"math/rand/v2"
	"sync"
)

var _ Registry = (*MockRegistry)(nil)

// MockRegistry implements Registry for testing purposes.
type MockRegistry struct {
	mu sync.Mutex

	NextIDCalled bool
	NextIDValue  uint32

	RegisterCalled   bool
	FunctionsStore   map[uint32]Functions
	UnregisterCalled bool
	GetCalled        bool
}

func (m *MockRegistry) Register(resource string, fns Functions) uint32 {
	id := rand.Uint32()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RegisterCalled = true
	if m.FunctionsStore == nil {
		m.FunctionsStore = make(map[uint32]Functions)
	}
	m.FunctionsStore[id] = fns
	return id
}

func (m *MockRegistry) Unregister(id uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnregisterCalled = true
	if m.FunctionsStore != nil {
		delete(m.FunctionsStore, id)
	}
}

func (m *MockRegistry) Get(id uint32) (ResourceFunction, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetCalled = true
	if m.FunctionsStore == nil {
		return ResourceFunction{}, false
	}
	fns, ok := m.FunctionsStore[id]
	return ResourceFunction{Functions: fns}, ok
}
