package xdgbase

import (
	"os"
	"path/filepath"
)

// DataHome returns the value of XDG_DATA_HOME, or the specification-defined
// fallback value of $HOME/.local/share.
func DataHome() string {
	return envSingle("XDG_DATA_HOME", func() string {
		return filepath.Join(home(), ".local", "share")
	})
}

// OtherDataDirs returns the values from XDG_DATA_DIRS, or the specification-defined
// fallback values "/usr/local/share/" and "/usr/share/".
func OtherDataDirs() []string {
	return envMulti("XDG_DATA_DIRS", func() []string {
		return []string{"/usr/local/share/", "/usr/share/"}
	})
}

// DataDirs returns the combination of DataHome and OtherDataDirs, giving the
// full set of data directories to search, in preference order.
func DataDirs() []string {
	ret := make([]string, 0, 3) // default OtherDataDirs has two elements
	ret = append(ret, DataHome())
	ret = append(ret, OtherDataDirs()...)
	return ret[:len(ret):len(ret)]
}

// ConfigHome returns the value of XDG_CONFIG_HOME, or the specification-defined
// fallback value of $HOME/.config.
func ConfigHome() string {
	return envSingle("XDG_CONFIG_HOME", func() string {
		return filepath.Join(home(), ".config")
	})
}

// OtherConfigDirs returns the values from XDG_CONFIG_DIRS, or the
// specification-defined fallback value "/etc/xdg".
func OtherConfigDirs() []string {
	return envMulti("XDG_CONFIG_DIRS", func() []string {
		return []string{"/etc/xdg"}
	})
}

// ConfigDirs returns the combination of ConfigHome and OtherConfigDirs, giving the
// full set of config directories to search, in preference order.
func ConfigDirs() []string {
	ret := make([]string, 0, 2) // default OtherConfigDirs has one element
	ret = append(ret, ConfigHome())
	ret = append(ret, OtherConfigDirs()...)
	return ret[:len(ret):len(ret)]
}

// CacheHome returns the value of XDG_CACHE_HOME, or the specification-defined
// fallback value of $HOME/.cache.
func CacheHome() string {
	return envSingle("XDG_CACHE_HOME", func() string {
		return filepath.Join(home(), ".cache")
	})
}

// MaybeRuntimeDir returns the value of XDG_RUNTIME_DIR, or an empty string if
// it is not set.
//
// Calling applications MUST check that the return value is non-empty before
// using it, because there is no reasonable default behavior when no runtime
// directory is defined.
func MaybeRuntimeDir() string {
	return envSingle("XDG_RUNTIME_DIR", func() string {
		return ""
	})
}

func envSingle(name string, fallback func() string) string {
	if p := os.Getenv(name); p != "" {
		if filepath.IsAbs(p) {
			return p
		}
	}

	return fallback()
}

func envMulti(name string, fallback func() []string) []string {
	if p := os.Getenv(name); p != "" {
		parts := filepath.SplitList(p)
		// Make sure all of the paths are absolute
		for i := len(parts) - 1; i >= 0; i-- {
			if !filepath.IsAbs(parts[i]) {
				// We'll shift everything after this point in the list
				// down so that this element is no longer present.
				copy(parts[i:], parts[i+1:])
				parts = parts[:len(parts)-1]
			}
		}
		parts = parts[:len(parts):len(parts)] // hide any extra capacity from the caller
		return parts
	}

	return fallback()
}
