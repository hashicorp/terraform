package state

import (
	"os"

	"github.com/hashicorp/terraform/terraform"
)

// LocalState manages a state storage that is local to the filesystem.
type LocalState struct {
	Path string

	state *terraform.State
}

// StateReader impl.
func (s *LocalState) State() *terraform.State {
	return s.state
}

// WriteState for LocalState always persists the state as well.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *terraform.State) error {
	s.state = state

	f, err := os.Create(s.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	return terraform.WriteState(s.state, f)
}

// PersistState for LocalState is a no-op since WriteState always persists.
//
// StatePersister impl.
func (s *LocalState) PersistState() error {
	return nil
}

// StateRefresher impl.
func (s *LocalState) RefreshState() error {
	f, err := os.Open(s.Path)
	if err != nil {
		// It is okay if the file doesn't exist, we treat that as a nil state
		if os.IsNotExist(err) {
			s.state = nil
			return nil
		}

		return err
	}
	defer f.Close()

	state, err := terraform.ReadState(f)
	if err != nil {
		return err
	}

	s.state = state
	return nil
}
