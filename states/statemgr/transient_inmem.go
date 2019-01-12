package statemgr

import (
	"sync"

	"github.com/hashicorp/terraform/states"
)

// NewTransientInMemory returns a Transient implementation that retains
// transient snapshots only in memory, as part of the object.
//
// The given initial state, if any, must not be modified concurrently while
// this function is running, but may be freely modified once this function
// returns without affecting the stored transient snapshot.
func NewTransientInMemory(initial *states.State) Transient {
	return &transientInMemory{
		current: initial.DeepCopy(),
	}
}

type transientInMemory struct {
	lock    sync.RWMutex
	current *states.State
}

var _ Transient = (*transientInMemory)(nil)

func (m *transientInMemory) State() *states.State {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.current.DeepCopy()
}

func (m *transientInMemory) WriteState(new *states.State) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.current = new.DeepCopy()
	return nil
}
