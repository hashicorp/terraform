package command

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/kardianos/osext"
)

// NOTE WELL: The logic in this file is primarily about plugin types OTHER THAN
// providers, which use an older set of approaches implemented here.
//
// The provider-related functions live primarily in meta_providers.go, and
// lean on some different underlying mechanisms in order to support automatic
// installation and a hierarchical addressing namespace, neither of which
// are supported for other plugin types.

// store the user-supplied path for plugin discovery
func (m *Meta) storePluginPath(pluginPath []string) error {
	if len(pluginPath) == 0 {
		return nil
	}

	m.fixupMissingWorkingDir()

	// remove the plugin dir record if the path was set to an empty string
	if len(pluginPath) == 1 && (pluginPath[0] == "") {
		return m.WorkingDir.SetForcedPluginDirs(nil)
	}

	return m.WorkingDir.SetForcedPluginDirs(pluginPath)
}

// Load the user-defined plugin search path into Meta.pluginPath if the file
// exists.
func (m *Meta) loadPluginPath() ([]string, error) {
	m.fixupMissingWorkingDir()
	return m.WorkingDir.ForcedPluginDirs()
}

// the default location for automatically installed plugins
func (m *Meta) pluginDir() string {
	return filepath.Join(m.DataDir(), "plugins", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
}

// pluginDirs return a list of directories to search for plugins.
//
// Earlier entries in this slice get priority over later when multiple copies
// of the same plugin version are found, but newer versions always override
// older versions where both satisfy the provider version constraints.
func (m *Meta) pluginDirs(includeAutoInstalled bool) []string {
	// user defined paths take precedence
	if len(m.pluginPath) > 0 {
		return m.pluginPath
	}

	// When searching the following directories, earlier entries get precedence
	// if the same plugin version is found twice, but newer versions will
	// always get preference below regardless of where they are coming from.
	// TODO: Add auto-install dir, default vendor dir and optional override
	// vendor dir(s).
	dirs := []string{"."}

	// Look in the same directory as the Terraform executable.
	// If found, this replaces what we found in the config path.
	exePath, err := osext.Executable()
	if err != nil {
		log.Printf("[ERROR] Error discovering exe directory: %s", err)
	} else {
		dirs = append(dirs, filepath.Dir(exePath))
	}

	// add the user vendor directory
	dirs = append(dirs, DefaultPluginVendorDir)

	if includeAutoInstalled {
		dirs = append(dirs, m.pluginDir())
	}
	dirs = append(dirs, m.GlobalPluginDirs...)

	return dirs
}
