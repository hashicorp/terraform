// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package workdir

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

const PluginPathFilename = "plugin_path"

// ProviderLocalCacheDir returns the directory we'll use as the
// working-directory-specific local cache of providers.
//
// The provider installer's job is to make sure that all providers needed for
// a particular working directory are available in this cache directory. No
// other component may write here, and in particular a Dir object itself
// never reads or writes into this directory, instead just delegating all of
// that responsibility to other components.
//
// Typically, the caller will ultimately pass the result of this method either
// directly or indirectly into providercache.NewDir, to get an object
// responsible for managing the contents.
func (d *Dir) ProviderLocalCacheDir() string {
	return filepath.Join(d.dataDir, "providers")
}

// ForcedPluginDirs returns a list of directories to use to find plugins,
// instead of the default locations.
//
// Returns an zero-length list and no error in the normal case where there
// are no overridden search directories. If ForcedPluginDirs returns a
// non-empty list with no errors then the result totally replaces the default
// search directories.
func (d *Dir) ForcedPluginDirs() ([]string, error) {
	raw, err := ioutil.ReadFile(filepath.Join(d.dataDir, PluginPathFilename))
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var pluginPath []string
	if err := json.Unmarshal(raw, &pluginPath); err != nil {
		return nil, err
	}
	return pluginPath, nil
}

// SetForcedPluginDirs records an overridden list of directories to search
// to find plugins, instead of the default locations. See ForcePluginDirs
// for more information.
//
// Pass a zero-length list to deactivate forced plugin directories altogether,
// thus allowing the working directory to return to using the default
// search directories.
func (d *Dir) SetForcedPluginDirs(dirs []string) error {

	filePath := filepath.Join(d.dataDir, PluginPathFilename)
	switch {
	case len(dirs) == 0:
		err := os.Remove(filePath)
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	default:
		// We'll ignore errors from this one, because if we fail to create
		// the directory then we'll fail to create the file below too,
		// and that subsequent error will more directly reflect what we
		// are trying to do here.
		d.ensureDataDir()

		raw, err := json.MarshalIndent(dirs, "", "  ")
		if err != nil {
			return err
		}

		return ioutil.WriteFile(filePath, raw, 0644)
	}
}
