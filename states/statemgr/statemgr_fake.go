package statemgr

import (
	"errors"
	"sync"

	"github.com/hashicorp/terraform/states"
)

// NewFullFake returns a full state manager that really only supports transient
// snapshots. This is primarily intended for testing and is not suitable for
// general use.
//
// The persistent part of the interface is stubbed out as an in-memory store,
// and so its snapshots are effectively also transient.
//
// The given Transient implementation is used to implement the transient
// portion of the interface. If nil is given, NewTransientInMemory is
// automatically called to create an in-memory transient manager with no
// initial transient snapshot.
//
// If the given initial state is non-nil then a copy of it will be used as
// the initial persistent snapshot.
//
// The Locker portion of the returned manager uses a local mutex to simulate
// mutually-exclusive access to the fake persistent portion of the object.
func NewFullFake(t Transient, initial *states.State) Full {
	if t == nil {
		t = NewTransientInMemory(nil)
	}

	// The "persistent" part of our manager is actually just another in-memory
	// transient used to fake a secondary storage layer.
	fakeP := NewTransientInMemory(initial.DeepCopy())

	return &fakeFull{
		t:     t,
		fakeP: fakeP,
	}
}

type fakeFull struct {
	t     Transient
	fakeP Transient

	lockLock sync.Mutex
	locked   bool
}

var _ Full = (*fakeFull)(nil)

func (m *fakeFull) State() *states.State {
	return m.t.State()
}

func (m *fakeFull) WriteState(s *states.State) error {
	return m.t.WriteState(s)
}

func (m *fakeFull) RefreshState() error {
	return m.t.WriteState(m.fakeP.State())
}

func (m *fakeFull) PersistState() error {
	return m.fakeP.WriteState(m.t.State())
}

func (m *fakeFull) Lock(info *LockInfo) (string, error) {
	m.lockLock.Lock()
	defer m.lockLock.Unlock()

	if m.locked {
		return "", &LockError{
			Err:  errors.New("fake state manager is locked"),
			Info: info,
		}
	}

	m.locked = true
	return "placeholder", nil
}

func (m *fakeFull) Unlock(id string) error {
	m.lockLock.Lock()
	defer m.lockLock.Unlock()

	if !m.locked {
		return errors.New("fake state manager is not locked")
	}
	if id != "placeholder" {
		return errors.New("wrong lock id for fake state manager")
	}

	m.locked = false
	return nil
}
