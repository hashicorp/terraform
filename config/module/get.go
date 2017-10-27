package module

import (
	"io/ioutil"
	"os"

	"github.com/hashicorp/go-getter"
)

// GetMode is an enum that describes how modules are loaded.
//
// GetModeLoad says that modules will not be downloaded or updated, they will
// only be loaded from the storage.
//
// GetModeGet says that modules can be initially downloaded if they don't
// exist, but otherwise to just load from the current version in storage.
//
// GetModeUpdate says that modules should be checked for updates and
// downloaded prior to loading. If there are no updates, we load the version
// from disk, otherwise we download first and then load.
type GetMode byte

const (
	GetModeNone GetMode = iota
	GetModeGet
	GetModeUpdate
)

// GetCopy is the same as Get except that it downloads a copy of the
// module represented by source.
//
// This copy will omit and dot-prefixed files (such as .git/, .hg/) and
// can't be updated on its own.
func GetCopy(dst, src string) error {
	// Create the temporary directory to do the real Get to
	tmpDir, err := ioutil.TempDir("", "tf")
	if err != nil {
		return err
	}
	// FIXME: This isn't completely safe. Creating and removing our temp path
	//        exposes where to race to inject files.
	if err := os.RemoveAll(tmpDir); err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Get to that temporary dir
	if err := getter.Get(tmpDir, src); err != nil {
		return err
	}

	// Make sure the destination exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Copy to the final location
	return copyDir(dst, tmpDir)
}
