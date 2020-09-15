// +build linux aix dragonfly freebsd netbsd openbsd solaris

package userdirs

import (
	"path/filepath"
	"strings"

	"github.com/apparentlymart/go-userdirs/xdgbase"
)

func supportedOS() bool {
	return true
}

func forApp(name string, vendor string, bundleID string) Dirs {
	// We use XDG conventions on Linux and other Unixes without their own special rules

	subDir := appDirName(name)

	ret := xdgDirs()
	for i, dir := range ret.ConfigDirs {
		ret.ConfigDirs[i] = filepath.Join(dir, subDir)
	}
	for i, dir := range ret.DataDirs {
		ret.DataDirs[i] = filepath.Join(dir, subDir)
	}
	ret.CacheDir = filepath.Join(ret.CacheDir, subDir)
	return ret
}

func xdgDirs() Dirs {
	return Dirs{
		ConfigDirs: xdgbase.ConfigDirs(),
		DataDirs:   xdgbase.DataDirs(),
		CacheDir:   xdgbase.CacheHome(),
	}
}

func appDirName(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}
