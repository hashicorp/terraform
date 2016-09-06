package state

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/terraform"
)

// LocalState manages a state storage that is local to the filesystem.
type LocalState struct {
	// Path is the path to read the state from. PathOut is the path to
	// write the state to. If PathOut is not specified, Path will be used.
	// If PathOut already exists, it will be overwritten.
	Path    string
	PathOut string

	state     *terraform.State
	readState *terraform.State
	written   bool
}

// SetState will force a specific state in-memory for this local state.
func (s *LocalState) SetState(state *terraform.State) {
	s.state = state
	s.readState = state
}

// StateReader impl.
func (s *LocalState) State() *terraform.State {
	return s.state.DeepCopy()
}

// WriteState for LocalState always persists the state as well.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *terraform.State) error {
	s.state = state

	path := s.PathOut
	if path == "" {
		path = s.Path
	}

	// If we don't have any state, we actually delete the file if it exists
	if state == nil {
		err := os.Remove(path)
		if err != nil && os.IsNotExist(err) {
			return nil
		}

		return err
	}

	// Create all the directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s.state.IncrementSerialMaybe(s.readState)
	s.readState = s.state

	if err := terraform.WriteState(s.state, f); err != nil {
		return err
	}

	s.written = true
	return nil
}

// PersistState for LocalState is a no-op since WriteState always persists.
//
// StatePersister impl.
func (s *LocalState) PersistState() error {
	return nil
}

// StateRefresher impl.
func (s *LocalState) RefreshState() error {
	// If we've never loaded before, read from Path, otherwise we
	// read from PathOut.
	path := s.Path
	if s.written && s.PathOut != "" {
		path = s.PathOut
	}

	f, err := os.Open(path)
	if err != nil {
		// It is okay if the file doesn't exist, we treat that as a nil state
		if !os.IsNotExist(err) {
			return err
		}

		f = nil
	}

	var state *terraform.State
	if f != nil {
		defer f.Close()
		state, err = terraform.ReadState(f)
		if err != nil {
			return err
		}
	}

	s.state = state
	s.readState = state
	return nil
}
