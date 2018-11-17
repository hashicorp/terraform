package statemgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

	// backupPath is an optional extra path which, if non-empty, will be
	// created or overwritten with the first state snapshot we read if there
	// is a subsequent call to write a different state.
	backupPath string

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

	file          *statefile.File
	readFile      *statefile.File
	backupFile    *statefile.File
	writtenBackup bool
}

var (
	_ Full           = (*Filesystem)(nil)
	_ PersistentMeta = (*Filesystem)(nil)
	_ Migrator       = (*Filesystem)(nil)
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

// SetBackupPath configures the receiever so that it will create a local
// backup file of the next state snapshot it reads (in State) if a different
// snapshot is subsequently written (in WriteState). Only one backup is
// written for the lifetime of the object, unless reset as described below.
//
// For correct operation, this must be called before any other state methods
// are called. If called multiple times, each call resets the backup
// function so that the next read will become the backup snapshot and a
// following write will save a backup of it.
func (s *Filesystem) SetBackupPath(path string) {
	s.backupPath = path
	s.backupFile = nil
	s.writtenBackup = false
}

// BackupPath returns the manager's backup path if backup files are enabled,
// or an empty string otherwise.
func (s *Filesystem) BackupPath() string {
	return s.backupPath
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
func (s *Filesystem) WriteState(state *states.State) error {
	// TODO: this should use a more robust method of writing state, by first
	// writing to a temp file on the same filesystem, and renaming the file over
	// the original.

	defer s.mutex()()

	if s.readFile == nil {
		err := s.refreshState()
		if err != nil {
			return err
		}
	}

	return s.writeState(state, nil)
}

func (s *Filesystem) writeState(state *states.State, meta *SnapshotMeta) error {
	if s.stateFileOut == nil {
		if err := s.createStateFiles(); err != nil {
			return nil
		}
	}
	defer s.stateFileOut.Sync()

	// We'll try to write our backup first, so we can be sure we've created
	// it successfully before clobbering the original file it came from.
	if !s.writtenBackup && s.backupFile != nil && s.backupPath != "" {
		if !statefile.StatesMarshalEqual(state, s.backupFile.State) {
			log.Printf("[TRACE] statemgr.Filesystem: creating backup snapshot at %s", s.backupPath)
			bfh, err := os.Create(s.backupPath)
			if err != nil {
				return fmt.Errorf("failed to create local state backup file: %s", err)
			}
			defer bfh.Close()

			err = statefile.Write(s.backupFile, bfh)
			if err != nil {
				return fmt.Errorf("failed to write to local state backup file: %s", err)
			}

			s.writtenBackup = true
		} else {
			log.Print("[TRACE] statemgr.Filesystem: not making a backup, because the new snapshot is identical to the old")
		}
	} else {
		// This branch is all just logging, to help understand why we didn't make a backup.
		switch {
		case s.backupPath == "":
			log.Print("[TRACE] statemgr.Filesystem: state file backups are disabled")
		case s.writtenBackup:
			log.Printf("[TRACE] statemgr.Filesystem: have already backed up original %s to %s on a previous write", s.path, s.backupPath)
		case s.backupFile == nil:
			log.Printf("[TRACE] statemgr.Filesystem: no original state snapshot to back up")
		default:
			log.Printf("[TRACE] statemgr.Filesystem: not creating a backup for an unknown reason")
		}
	}

	s.file = s.file.DeepCopy()
	if s.file == nil {
		s.file = NewStateFile()
	}
	s.file.State = state.DeepCopy()

	if _, err := s.stateFileOut.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	if err := s.stateFileOut.Truncate(0); err != nil {
		return err
	}

	if state == nil {
		// if we have no state, don't write anything else.
		log.Print("[TRACE] statemgr.Filesystem: state is nil, so leaving the file empty")
		return nil
	}

	if meta == nil {
		if s.readFile == nil || !statefile.StatesMarshalEqual(s.file.State, s.readFile.State) {
			s.file.Serial++
			log.Printf("[TRACE] statemgr.Filesystem: state has changed since last snapshot, so incrementing serial to %d", s.file.Serial)
		} else {
			log.Print("[TRACE] statemgr.Filesystem: no state changes since last snapshot")
		}
	} else {
		// Force new metadata
		s.file.Lineage = meta.Lineage
		s.file.Serial = meta.Serial
		log.Printf("[TRACE] statemgr.Filesystem: forcing lineage %q serial %d for migration/import", s.file.Lineage, s.file.Serial)
	}

	log.Printf("[TRACE] statemgr.Filesystem: writing snapshot at %s", s.path)
	if err := statefile.Write(s.file, s.stateFileOut); err != nil {
		return err
	}

	// Any future reads must come from the file we've now updated
	s.readPath = s.path
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
	return s.refreshState()
}

func (s *Filesystem) refreshState() error {
	var reader io.Reader

	// The s.readPath file is only OK to read if we have not written any state out
	// (in which case the same state needs to be read in), and no state output file
	// has been opened (possibly via a lock) or the input path is different
	// than the output path.
	// This is important for Windows, as if the input file is the same as the
	// output file, and the output file has been locked already, we can't open
	// the file again.
	if s.stateFileOut == nil || s.readPath != s.path {
		// we haven't written a state file yet, so load from readPath
		log.Printf("[TRACE] statemgr.Filesystem: reading initial snapshot from %s", s.readPath)
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
		log.Printf("[TRACE] statemgr.Filesystem: reading latest snapshot from %s", s.path)
		// no state to refresh
		if s.stateFileOut == nil {
			return nil
		}

		// we have a state file, make sure we're at the start
		s.stateFileOut.Seek(0, os.SEEK_SET)
		reader = s.stateFileOut
	}

	f, err := statefile.Read(reader)
	// if there's no state then a nil file is fine
	if err != nil {
		if err != statefile.ErrNoState {
			return err
		}
		log.Printf("[TRACE] statemgr.Filesystem: snapshot file has nil snapshot, but that's okay")
	}

	s.file = f
	s.readFile = s.file.DeepCopy()
	if s.file != nil {
		log.Printf("[TRACE] statemgr.Filesystem: read snapshot with lineage %q serial %d", s.file.Lineage, s.file.Serial)
	} else {
		log.Print("[TRACE] statemgr.Filesystem: read nil snapshot")
	}
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

	lockInfoPath := s.lockInfoPath()
	log.Printf("[TRACE] statemgr.Filesystem: removing lock metadata file %s", lockInfoPath)
	os.Remove(lockInfoPath)

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

// StateForMigration is part of our implementation of Migrator.
func (s *Filesystem) StateForMigration() *statefile.File {
	return s.file.DeepCopy()
}

// WriteStateForMigration is part of our implementation of Migrator.
func (s *Filesystem) WriteStateForMigration(f *statefile.File, force bool) error {
	defer s.mutex()()

	if s.readFile == nil {
		err := s.refreshState()
		if err != nil {
			return err
		}
	}

	if !force {
		err := CheckValidImport(f, s.readFile)
		if err != nil {
			return err
		}
	}

	if s.readFile != nil {
		log.Printf(
			"[TRACE] statemgr.Filesystem: Importing snapshot with lineage %q serial %d over snapshot with lineage %q serial %d at %s",
			f.Lineage, f.Serial,
			s.readFile.Lineage, s.readFile.Serial,
			s.path,
		)
	} else {
		log.Printf(
			"[TRACE] statemgr.Filesystem: Importing snapshot with lineage %q serial %d as the initial state snapshot at %s",
			f.Lineage, f.Serial,
			s.path,
		)
	}

	err := s.writeState(f.State, &SnapshotMeta{Lineage: f.Lineage, Serial: f.Serial})
	if err != nil {
		return err
	}

	return nil
}

// Open the state file, creating the directories and file as needed.
func (s *Filesystem) createStateFiles() error {
	log.Printf("[TRACE] statemgr.Filesystem: preparing to manage state snapshots at %s", s.path)

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

	// If the file already existed with content then that'll be the content
	// of our backup file if we write a change later.
	s.backupFile, err = statefile.Read(s.stateFileOut)
	if err != nil {
		if err != statefile.ErrNoState {
			return err
		}
		log.Printf("[TRACE] statemgr.Filesystem: no previously-stored snapshot exists")
	} else {
		log.Printf("[TRACE] statemgr.Filesystem: existing snapshot has lineage %q serial %d", s.backupFile.Lineage, s.backupFile.Serial)
	}

	// Refresh now, to load in the snapshot if the file already existed
	return nil
}

// return the path for the lockInfo metadata.
func (s *Filesystem) lockInfoPath() string {
	stateDir, stateName := filepath.Split(s.path)
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

	log.Printf("[TRACE] statemgr.Filesystem: writing lock metadata to %s", path)
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
