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
	// Extra info passed to State.Lock
	Reason string
}

// return the lock info formatted in an error
func (l *lockInfo) Err() error {
	return fmt.Errorf("state file %q locked. created:%s, reason:%s",
		l.Path, l.Created, l.Reason)
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
	// created is set to tru if stateFileOut didn't exist before we created it.
	// This is mostly so we can clean up emtpy files during tests, but doesn't
	// hurt to remove file we never wrote to.
	created bool

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
	// we can't be locked if we don't have a file
	if s.stateFileOut == nil {
		return nil
	}

	os.Remove(s.lockInfoPath())

	fileName := s.stateFileOut.Name()

	unlockErr := s.unlock()
	s.stateFileOut.Close()
	s.stateFileOut = nil

	// clean up the state file if we created it an never wrote to it
	stat, err := os.Stat(fileName)
	if err == nil && stat.Size() == 0 && s.created {
		os.Remove(fileName)
	}

	return unlockErr
}

// Open the state file, creating the directories and file as needed.
func (s *LocalState) createStateFiles() error {
	if s.PathOut == "" {
		s.PathOut = s.Path
	}

	// yes this could race, but we only use it to clean up empty files
	if _, err := os.Stat(s.PathOut); os.IsNotExist(err) {
		s.created = true
	}

	// Create all the directories
	if err := os.MkdirAll(filepath.Dir(s.PathOut), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(s.PathOut, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	s.stateFileOut = f
	return nil
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
