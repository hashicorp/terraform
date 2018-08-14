package state

import (
	"sync"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
)

// BackupState wraps a State that backs up the state on the first time that
// a WriteState or PersistState is called.
//
// If Path exists, it will be overwritten.
type BackupState struct {
	mu   sync.Mutex
	Real State
	Path string

	done bool
}

func (s *BackupState) State() *states.State {
	return s.Real.State()
}

func (s *BackupState) RefreshState() error {
	return s.Real.RefreshState()
}

func (s *BackupState) WriteState(state *states.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.done {
		if err := s.backup(); err != nil {
			return err
		}
	}

	return s.Real.WriteState(state)
}

func (s *BackupState) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.done {
		if err := s.backup(); err != nil {
			return err
		}
	}

	return s.Real.PersistState()
}

func (s *BackupState) Lock(info *LockInfo) (string, error) {
	return s.Real.Lock(info)
}

func (s *BackupState) Unlock(id string) error {
	return s.Real.Unlock(id)
}

func (s *BackupState) backup() error {
	state := s.Real.State()
	if state == nil {
		if err := s.Real.RefreshState(); err != nil {
			return err
		}

		state = s.Real.State()
	}

	// LocalState.WriteState ensures that a file always exists for locking
	// purposes, but we don't need a backup or lock if the state is empty, so
	// skip this with a nil state.
	if state != nil {
		ls := statemgr.NewFilesystem(s.Path)
		if err := ls.WriteState(state); err != nil {
			return err
		}
	}

	s.done = true
	return nil
}
