// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package clistate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/states/statemgr"
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
	// This is mostly so we can clean up empty files during tests, but doesn't
	// hurt to remove file we never wrote to.
	created bool

	state     *workdir.BackendStateFile
	readState *workdir.BackendStateFile
	written   bool
}

// SetState will force a specific state in-memory for this local state.
func (s *LocalState) SetState(state *workdir.BackendStateFile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = state.DeepCopy()
	s.readState = state.DeepCopy()
}

// StateReader impl.
func (s *LocalState) State() *workdir.BackendStateFile {
	return s.state.DeepCopy()
}

// WriteState for LocalState always persists the state as well.
// TODO: this should use a more robust method of writing state, by first
// writing to a temp file on the same filesystem, and renaming the file over
// the original.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *workdir.BackendStateFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return nil
		}
	}
	defer s.stateFileOut.Sync()

	s.state = state.DeepCopy() // don't want mutations before we actually get this written to disk

	if _, err := s.stateFileOut.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := s.stateFileOut.Truncate(0); err != nil {
		return err
	}

	if state == nil {
		// if we have no state, don't write anything else.
		return nil
	}

	raw, err := workdir.EncodeBackendStateFile(state)
	if err != nil {
		return err
	}
	_, err = s.stateFileOut.Write(raw)
	if err != nil {
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

	if s.PathOut == "" {
		s.PathOut = s.Path
	}

	var reader io.Reader

	// The s.Path file is only OK to read if we have not written any state out
	// (in which case the same state needs to be read in), and no state output file
	// has been opened (possibly via a lock) or the input path is different
	// than the output path.
	// This is important for Windows, as if the input file is the same as the
	// output file, and the output file has been locked already, we can't open
	// the file again.
	if !s.written && (s.stateFileOut == nil || s.Path != s.PathOut) {
		// we haven't written a state file yet, so load from Path
		f, err := os.Open(s.Path)
		if err != nil {
			// It is okay if the file doesn't exist, we treat that as a nil state
			if !os.IsNotExist(err) {
				return err
			}

			// a nil reader means no state at all, handled below
			reader = nil

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
		s.stateFileOut.Seek(0, io.SeekStart)
		reader = s.stateFileOut
	}

	var state *workdir.BackendStateFile
	if reader != nil { // otherwise we'll leave state as nil
		raw, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		state, err = workdir.ParseBackendStateFile(raw)
		if err != nil {
			return err
		}
	}

	s.state = state
	s.readState = s.state.DeepCopy()
	return nil
}

// Lock implements a local filesystem state.Locker.
func (s *LocalState) Lock(info *statemgr.LockInfo) (string, error) {
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
			err = errors.Join(err, infoErr)
		}

		lockErr := &statemgr.LockError{
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
			idErr = errors.Join(idErr, err)
		}

		return &statemgr.LockError{
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
func (s *LocalState) lockInfo() (*statemgr.LockInfo, error) {
	path := s.lockInfoPath()
	infoData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := statemgr.LockInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but could not unmarshal lock info: %s", s.Path, err)
	}
	return &info, nil
}

// write a new lock info file
func (s *LocalState) writeLockInfo(info *statemgr.LockInfo) error {
	path := s.lockInfoPath()
	info.Path = s.Path
	info.Created = time.Now().UTC()

	err := ioutil.WriteFile(path, info.Marshal(), 0600)
	if err != nil {
		return fmt.Errorf("could not write lock info for %q: %s", s.Path, err)
	}
	return nil
}
