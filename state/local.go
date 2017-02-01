package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

// lock metadata structure for local locks
type lockInfo struct {
	// Path to the state file
	Path string
	// The time the lock was taken
	Created time.Time
	// The time this lock expires
	Expires time.Time
	// The lock reason passed to State.Lock
	Reason string
}

// return the lock info formatted in an error
func (l *lockInfo) Err() error {
	return fmt.Errorf("state file %q locked. created:%s, expires:%s, reason:%s",
		l.Path, l.Created, l.Expires, l.Reason)
}

// LocalState manages a state storage that is local to the filesystem.
type LocalState struct {
	// Path is the path to read the state from. PathOut is the path to
	// write the state to. If PathOut is not specified, Path will be used.
	// If PathOut already exists, it will be overwritten.
	Path    string
	PathOut string

	// the file handle corresponding to PathOut
	stateFileOut *os.File

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

// Lock implements a local filesystem state.Locker.
func (s *LocalState) Lock(reason string) error {
	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return err
		}
	}

	if err := s.lock(); err != nil {
		if info, err := s.lockInfo(); err == nil {
			return info.Err()
		}
		return fmt.Errorf("state file %q locked: %s", s.Path, err)
	}

	return s.writeLockInfo(reason)
}

func (s *LocalState) Unlock() error {
	os.Remove(s.lockInfoPath())
	return s.unlock()
}

// Open the state file, creating the directories and file as needed.
func (s *LocalState) createStateFiles() error {
	if s.PathOut == "" {
		s.PathOut = s.Path
	}

	f, err := createFileAndDirs(s.PathOut)
	if err != nil {
		return err
	}
	s.stateFileOut = f
	return nil
}

func createFileAndDirs(path string) (*os.File, error) {
	// Create all the directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// WriteState for LocalState always persists the state as well.
// TODO: this should use a more robust method of writing state, by first
// writing to a temp file on the same filesystem, and renaming the file over
// the original.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *terraform.State) error {
	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return nil
		}
	}
	defer s.stateFileOut.Sync()

	s.state = state

	if _, err := s.stateFileOut.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	if err := s.stateFileOut.Truncate(0); err != nil {
		return err
	}

	if state == nil {
		// if we have no state, don't write anything else.
		return nil
	}

	s.state.IncrementSerialMaybe(s.readState)
	s.readState = s.state

	if err := terraform.WriteState(s.state, s.stateFileOut); err != nil {
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
	var reader io.Reader
	if !s.written {
		// we haven't written a state file yet, so load from Path
		f, err := os.Open(s.Path)
		if err != nil {
			// It is okay if the file doesn't exist, we treat that as a nil state
			if !os.IsNotExist(err) {
				return err
			}

			// we need a non-nil reader for ReadState and an empty buffer works
			// to return EOF immediately
			reader = bytes.NewBuffer(nil)

		} else {
			defer f.Close()
			reader = f
		}
	} else {
		// we have a state file, make sure we're at the start
		s.stateFileOut.Seek(0, os.SEEK_SET)
		reader = s.stateFileOut
	}

	state, err := terraform.ReadState(reader)
	// if there's no state we just assign the nil return value
	if err != nil && err != terraform.ErrNoState {
		return err
	}

	s.state = state
	s.readState = state
	return nil
}

// return the path for the lockInfo metadata.
func (s *LocalState) lockInfoPath() string {
	stateDir, stateName := filepath.Split(s.Path)
	if stateName == "" {
		panic("empty state file path")
	}

	if stateName[0] == '.' {
		stateName = stateName[1:]
	}

	return filepath.Join(stateDir, fmt.Sprintf(".%s.lock.info", stateName))
}

// lockInfo returns the data in a lock info file
func (s *LocalState) lockInfo() (*lockInfo, error) {
	path := s.lockInfoPath()
	infoData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := lockInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but could not unmarshal lock info: %s", s.Path, err)
	}
	return &info, nil
}

// write a new lock info file
func (s *LocalState) writeLockInfo(reason string) error {
	path := s.lockInfoPath()

	lockInfo := &lockInfo{
		Path:    s.Path,
		Created: time.Now().UTC(),
		Expires: time.Now().Add(time.Hour).UTC(),
		Reason:  reason,
	}

	infoData, err := json.Marshal(lockInfo)
	if err != nil {
		panic(fmt.Sprintf("could not marshal lock info: %#v", lockInfo))
	}

	err = ioutil.WriteFile(path, infoData, 0600)
	if err != nil {
		return fmt.Errorf("could not write lock info for %q: %s", s.Path, err)
	}
	return nil
}
