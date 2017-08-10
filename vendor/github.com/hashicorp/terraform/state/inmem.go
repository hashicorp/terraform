package state

import (
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

// InmemState is an in-memory state storage.
type InmemState struct {
	mu    sync.Mutex
	state *terraform.State
}

func (s *InmemState) State() *terraform.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

func (s *InmemState) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return nil
}

func (s *InmemState) WriteState(state *terraform.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state = state.DeepCopy()

	if s.state != nil {
		state.Serial = s.state.Serial

		if !s.state.MarshalEqual(state) {
			state.Serial++
		}
	}

	s.state = state

	return nil
}

func (s *InmemState) PersistState() error {
	return nil
}

func (s *InmemState) Lock(*LockInfo) (string, error) {
	return "", nil
}

func (s *InmemState) Unlock(string) error {
	return nil
}

// inmemLocker is an in-memory State implementation for testing locks.
type inmemLocker struct {
	*InmemState

	mu       sync.Mutex
	lockInfo *LockInfo
	// count the calls to Lock
	lockCounter int
}

func (s *inmemLocker) Lock(info *LockInfo) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockCounter++

	lockErr := &LockError{
		Info: &LockInfo{},
	}

	if s.lockInfo != nil {
		lockErr.Err = errors.New("state locked")
		*lockErr.Info = *s.lockInfo
		return "", lockErr
	}

	info.Created = time.Now().UTC()
	s.lockInfo = info
	return s.lockInfo.ID, nil
}

func (s *inmemLocker) Unlock(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lockErr := &LockError{
		Info: &LockInfo{},
	}

	if id != s.lockInfo.ID {
		lockErr.Err = errors.New("invalid lock id")
		*lockErr.Info = *s.lockInfo
		return lockErr
	}

	s.lockInfo = nil
	return nil
}
