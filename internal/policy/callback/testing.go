// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import "sync"

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

func (m *MockRegistry) Register(id uint32, fns Functions) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RegisterCalled = true
	if m.FunctionsStore == nil {
		m.FunctionsStore = make(map[uint32]Functions)
	}
	m.FunctionsStore[id] = fns
}

func (m *MockRegistry) Unregister(id uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnregisterCalled = true
	if m.FunctionsStore != nil {
		delete(m.FunctionsStore, id)
	}
}

func (m *MockRegistry) Get(id uint32) (Functions, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetCalled = true
	if m.FunctionsStore == nil {
		return Functions{}, false
	}
	fns, ok := m.FunctionsStore[id]
	return fns, ok
}

func (m *MockRegistry) NextID() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NextIDCalled = true
	if m.NextIDValue != 0 {
		return m.NextIDValue
	}
	m.NextIDValue++
	return m.NextIDValue
}
