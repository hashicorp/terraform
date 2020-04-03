// +build windows

package userdirs

import (
	"path/filepath"

	"github.com/apparentlymart/go-userdirs/windowsbase"
)

func supportedOS() bool {
	return true
}

func forApp(name string, vendor string, bundleID string) Dirs {
	subDir := filepath.Join(vendor, name)
	localBase, err := windowsbase.LocalAppDataDir()
	if err != nil {
		localBase = ""
	}
	roamingBase, err := windowsbase.RoamingAppDataDir()
	if err != nil {
		roamingBase = ""
	}
	if localBase == "" {
		// Should never happen in practice, because this is always set on Windows
		localBase = "c:\\"
	}
	if roamingBase == "" {
		roamingBase = localBase // store everything locally, then
	}

	roamingDir := filepath.Join(roamingBase, subDir)
	localDir := filepath.Join(localBase, subDir)

	return Dirs{
		ConfigDirs: []string{roamingDir},
		DataDirs:   []string{roamingDir},
		CacheDir:   localDir,
	}
}
