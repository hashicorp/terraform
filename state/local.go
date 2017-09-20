package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/terraform"
)

// LocalState manages a state storage that is local to the filesystem.
type LocalState struct {
	mu sync.Mutex

	// Path is the path to read the state from. PathOut is the path to
	// write the state to. If PathOut is not specified, Path will be used.
	// If PathOut already exists, it will be overwritten.
	Path    string
	PathOut string

	// the file handle corresponding to PathOut
	stateFileOut *os.File

	// While the stateFileOut will correspond to the lock directly,
	// store and check the lock ID to maintain a strict state.Locker
	// implementation.
	lockID string

	// created is set to true if stateFileOut didn't exist before we created it.
	// This is mostly so we can clean up emtpy files during tests, but doesn't
	// hurt to remove file we never wrote to.
	created bool

	state     *terraform.State
	readState *terraform.State
	written   bool
}

// SetState will force a specific state in-memory for this local state.
func (s *LocalState) SetState(state *terraform.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = state.DeepCopy()
	s.readState = state.DeepCopy()
}

// StateReader impl.
func (s *LocalState) State() *terraform.State {
	return s.state.DeepCopy()
}

// WriteState for LocalState always persists the state as well.
// TODO: this should use a more robust method of writing state, by first
// writing to a temp file on the same filesystem, and renaming the file over
// the original.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *terraform.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return nil
		}
	}
	defer s.stateFileOut.Sync()

	s.state = state.DeepCopy() // don't want mutations before we actually get this written to disk

	if s.readState != nil && s.state != nil {
		// We don't trust callers to properly manage serials. Instead, we assume
		// that a WriteState is always for the next version after what was
		// most recently read.
		s.state.Serial = s.readState.Serial
	}

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

	if !s.state.MarshalEqual(s.readState) {
		s.state.Serial++
	}

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
	s.mu.Lock()
	defer s.mu.Unlock()

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
		// no state to refresh
		if s.stateFileOut == nil {
			return nil
		}

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
	s.readState = s.state.DeepCopy()
	return nil
}

// Lock implements a local filesystem state.Locker.
func (s *LocalState) Lock(info *LockInfo) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return "", err
		}
	}

	if s.lockID != "" {
		return "", fmt.Errorf("state %q already locked", s.stateFileOut.Name())
	}

	if err := s.lock(); err != nil {
		info, infoErr := s.lockInfo()
		if infoErr != nil {
			err = multierror.Append(err, infoErr)
		}

		lockErr := &LockError{
			Info: info,
			Err:  err,
		}

		return "", lockErr
	}

	s.lockID = info.ID
	return s.lockID, s.writeLockInfo(info)
}

func (s *LocalState) Unlock(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockID == "" {
		return fmt.Errorf("LocalState not locked")
	}

	if id != s.lockID {
		idErr := fmt.Errorf("invalid lock id: %q. current id: %q", id, s.lockID)
		info, err := s.lockInfo()
		if err != nil {
			err = multierror.Append(idErr, err)
		}

		return &LockError{
			Err:  idErr,
			Info: info,
		}
	}

	os.Remove(s.lockInfoPath())

	fileName := s.stateFileOut.Name()

	unlockErr := s.unlock()

	s.stateFileOut.Close()
	s.stateFileOut = nil
	s.lockID = ""

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
func (s *LocalState) lockInfo() (*LockInfo, error) {
	path := s.lockInfoPath()
	infoData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := LockInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but could not unmarshal lock info: %s", s.Path, err)
	}
	return &info, nil
}

// write a new lock info file
func (s *LocalState) writeLockInfo(info *LockInfo) error {
	path := s.lockInfoPath()
	info.Path = s.Path
	info.Created = time.Now().UTC()

	err := ioutil.WriteFile(path, info.Marshal(), 0600)
	if err != nil {
		return fmt.Errorf("could not write lock info for %q: %s", s.Path, err)
	}
	return nil
}
