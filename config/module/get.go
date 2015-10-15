package module

import (
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

func getStorage(s getter.Storage, key string, src string, mode GetMode) (string, bool, error) {
	// Get the module with the level specified if we were told to.
	if mode > GetModeNone {
		if err := s.Get(key, src, mode == GetModeUpdate); err != nil {
			return "", false, err
		}
	}

	// Get the directory where the module is.
	return s.Dir(key)
}
