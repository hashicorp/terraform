package statemgr

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

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

// Filesystem is a full state manager that uses a file in the local filesystem
// for persistent storage.
//
// The transient storage for Filesystem is always in-memory.
type Filesystem struct {
	mu sync.Mutex

	// path is the location where a file will be created or replaced for
	// each persistent snapshot.
	path string

	// readPath is read by RefreshState instead of "path" until the first
	// call to PersistState, after which it is ignored.
	//
	// The file at readPath must never be written to by this manager.
	readPath string

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

	file     *statefile.File
	readFile *statefile.File
	written  bool
}

var (
	_ Full           = (*Filesystem)(nil)
	_ PersistentMeta = (*Filesystem)(nil)
)

// NewFilesystem creates a filesystem-based state manager that reads and writes
// state snapshots at the given filesystem path.
//
// This is equivalent to calling NewFileSystemBetweenPaths with statePath as
// both of the path arguments.
func NewFilesystem(statePath string) *Filesystem {
	return &Filesystem{
		path:     statePath,
		readPath: statePath,
	}
}

// NewFilesystemBetweenPaths creates a filesystem-based state manager that
// reads an initial snapshot from readPath and then writes all new snapshots to
// writePath.
func NewFilesystemBetweenPaths(readPath, writePath string) *Filesystem {
	return &Filesystem{
		path:     writePath,
		readPath: readPath,
	}
}

// State is an implementation of Reader.
func (s *Filesystem) State() *states.State {
	defer s.mutex()()
	if s.file == nil {
		return nil
	}
	return s.file.DeepCopy().State
}

// WriteState is an incorrect implementation of Writer that actually also
// persists.
// WriteState for LocalState always persists the state as well.
//
// StateWriter impl.
func (s *Filesystem) WriteState(state *states.State) error {
	// TODO: this should use a more robust method of writing state, by first
	// writing to a temp file on the same filesystem, and renaming the file over
	// the original.

	defer s.mutex()()

	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return nil
		}
	}
	defer s.stateFileOut.Sync()

	s.file = s.file.DeepCopy()
	s.file.State = state.DeepCopy()

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

	if !statefile.StatesMarshalEqual(s.file.State, s.readFile.State) {
		s.file.Serial++
	}

	if err := statefile.Write(s.file, s.stateFileOut); err != nil {
		return err
	}

	s.written = true
	return nil
}

// PersistState is an implementation of Persister that does nothing because
// this type's Writer implementation does its own persistence.
func (s *Filesystem) PersistState() error {
	return nil
}

// RefreshState is an implementation of Refresher.
func (s *Filesystem) RefreshState() error {
	defer s.mutex()()

	var reader io.Reader

	// The s.readPath file is only OK to read if we have not written any state out
	// (in which case the same state needs to be read in), and no state output file
	// has been opened (possibly via a lock) or the input path is different
	// than the output path.
	// This is important for Windows, as if the input file is the same as the
	// output file, and the output file has been locked already, we can't open
	// the file again.
	if !s.written && (s.stateFileOut == nil || s.readPath != s.path) {
		// we haven't written a state file yet, so load from readPath
		f, err := os.Open(s.readPath)
		if err != nil {
			// It is okay if the file doesn't exist; we'll treat that as a nil state.
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

	f, err := statefile.Read(reader)
	// if there's no state we just assign the nil return value
	if err != nil && err != statefile.ErrNoState {
		return err
	}

	s.file = f
	s.readFile = s.file.DeepCopy()
	return nil
}

// Lock implements Locker using filesystem discretionary locks.
func (s *Filesystem) Lock(info *LockInfo) (string, error) {
	defer s.mutex()()

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

// Unlock is the companion to Lock, completing the implemention of Locker.
func (s *Filesystem) Unlock(id string) error {
	defer s.mutex()()

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

// StateSnapshotMeta returns the metadata from the most recently persisted
// or refreshed persistent state snapshot.
//
// This is an implementation of PersistentMeta.
func (s *Filesystem) StateSnapshotMeta() SnapshotMeta {
	if s.file == nil {
		return SnapshotMeta{} // placeholder
	}

	return SnapshotMeta{
		Lineage: s.file.Lineage,
		Serial:  s.file.Serial,

		TerraformVersion: s.file.TerraformVersion,
	}
}

// Open the state file, creating the directories and file as needed.
func (s *Filesystem) createStateFiles() error {

	// This could race, but we only use it to clean up empty files
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		s.created = true
	}

	// Create all the directories
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	s.stateFileOut = f
	return nil
}

// return the path for the lockInfo metadata.
func (s *Filesystem) lockInfoPath() string {
	stateDir, stateName := filepath.Split(s.readPath)
	if stateName == "" {
		panic("empty state file path")
	}

	if stateName[0] == '.' {
		stateName = stateName[1:]
	}

	return filepath.Join(stateDir, fmt.Sprintf(".%s.lock.info", stateName))
}

// lockInfo returns the data in a lock info file
func (s *Filesystem) lockInfo() (*LockInfo, error) {
	path := s.lockInfoPath()
	infoData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := LockInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but could not unmarshal lock info: %s", s.readPath, err)
	}
	return &info, nil
}

// write a new lock info file
func (s *Filesystem) writeLockInfo(info *LockInfo) error {
	path := s.lockInfoPath()
	info.Path = s.readPath
	info.Created = time.Now().UTC()

	err := ioutil.WriteFile(path, info.Marshal(), 0600)
	if err != nil {
		return fmt.Errorf("could not write lock info for %q: %s", s.readPath, err)
	}
	return nil
}

func (s *Filesystem) mutex() func() {
	s.mu.Lock()
	return s.mu.Unlock
}
