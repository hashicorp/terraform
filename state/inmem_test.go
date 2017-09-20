package state

import (
	"testing"
)

func TestInmemState(t *testing.T) {
	TestState(t, &InmemState{state: TestStateInitial()})
}

func TestInmemState_impl(t *testing.T) {
	var _ StateReader = new(InmemState)
	var _ StateWriter = new(InmemState)
	var _ StatePersister = new(InmemState)
	var _ StateRefresher = new(InmemState)
}

func TestInmemLocker(t *testing.T) {
	inmem := &InmemState{state: TestStateInitial()}
	// test that it correctly wraps the inmem state
	s := &inmemLocker{InmemState: inmem}
	TestState(t, s)

	info := NewLockInfo()

	id, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	if id == "" {
		t.Fatal("no lock id from state lock")
	}

	// locking again should fail
	_, err = s.Lock(NewLockInfo())
	if err == nil {
		t.Fatal("state locked while locked")
	}

	if err.(*LockError).Info.ID != id {
		t.Fatal("wrong lock id from lock failure")
	}

	if err := s.Unlock(id); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Lock(NewLockInfo()); err != nil {
		t.Fatal(err)
	}
}
