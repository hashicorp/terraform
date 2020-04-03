// +build darwin

package userdirs

import (
	"path/filepath"

	"github.com/apparentlymart/go-userdirs/macosbase"
)

func supportedOS() bool {
	return true
}

func forApp(name string, vendor string, bundleID string) Dirs {
	appSupportDir := filepath.Join(macosbase.ApplicationSupportDir(), bundleID)
	cachesDir := filepath.Join(macosbase.CachesDir(), bundleID)
	globalAppSupportDir := filepath.Join("/", "Library", "Application Support", bundleID)

	return Dirs{
		// NOTE: We don't use "Preferences" here because it is specified as
		// containing propertly list files managed by an OS framework API only,
		// so it would not be appropriate to read/write arbitrary config
		// files in there.
		ConfigDirs: []string{appSupportDir},
		DataDirs:   []string{appSupportDir, globalAppSupportDir},
		CacheDir:   cachesDir,
	}
}
