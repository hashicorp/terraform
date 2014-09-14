package module

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// FolderStorage is an implementation of the Storage interface that manages
// modules on the disk.
type FolderStorage struct {
	// StorageDir is the directory where the modules will be stored.
	StorageDir string
}

// Dir implements Storage.Dir
func (s *FolderStorage) Dir(source string) (d string, e bool, err error) {
	d = s.dir(source)
	_, err = os.Stat(d)
	if err == nil {
		// Directory exists
		e = true
		return
	}
	if os.IsNotExist(err) {
		// Directory doesn't exist
		d = ""
		e = false
		err = nil
		return
	}

	// An error
	d = ""
	e = false
	return
}

// Get implements Storage.Get
func (s *FolderStorage) Get(source string, update bool) error {
	dir := s.dir(source)
	if !update {
		if _, err := os.Stat(dir); err == nil {
			// If the directory already exists, then we're done since
			// we're not updating.
			return nil
		} else if !os.IsNotExist(err) {
			// If the error we got wasn't a file-not-exist error, then
			// something went wrong and we should report it.
			return fmt.Errorf("Error reading module directory: %s", err)
		}
	}

	// Get the source. This always forces an update.
	return Get(dir, source)
}

// dir returns the directory name internally that we'll use to map to
// internally.
func (s *FolderStorage) dir(source string) string {
	sum := md5.Sum([]byte(source))
	return filepath.Join(s.StorageDir, hex.EncodeToString(sum[:]))
}
