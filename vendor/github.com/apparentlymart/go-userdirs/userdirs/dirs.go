package userdirs

import (
	"os"
	"path/filepath"
)

// Dirs represents a set of directory paths with different purposes.
type Dirs struct {
	// ConfigDirs is a list, in preference order, of directory paths to search
	// for configuration files.
	//
	// The list must always contain at least one element, and its first element
	// is the directory where any new configuration files should be written.
	//
	// On some systems, ConfigDirs and DataDirs may overlap, so applications
	// which scan the contents of the configuration directories should impose
	// some additional filtering to distinguish configuration files from data
	// files.
	//
	// Files placed in ConfigDirs should ideally be things that it would be
	// reasonable to share among multiple systems (possibly on different
	// platforms, possibly to check into a version control system, etc.
	ConfigDirs []string

	// DataDirs is a list, in preference order, of directory paths to search for
	// data files.
	//
	// The list must always contain at least one element, and its first element
	// is the directory where any new data files should be written.
	//
	// On some systems, ConfigDirs and DataDirs may overlap, so applications
	// which scan the contents of the data directories should impose some
	// additional filtering to distinguish data files from configuration files.
	DataDirs []string

	// CacheDir is the path of a single directory that can be used for temporary
	// cache data.
	//
	// The cache is suitable only for data that the calling application could
	// recreate if lost. Any file or directory under this prefix may be deleted
	// at any time by other software.
	//
	// This directory may, on some systems, match one of the directories
	// returned in ConfigDirs and/or DataDirs. For this reason applications
	// must ensure that they do not misinterpret config and data files as
	// cache files, and in particular should not naively purge a cache by
	// emptying this directory.
	CacheDir string
}

// ConfigHome returns the path for the directory where any new configuration
// files should be written.
func (d Dirs) ConfigHome() string {
	return d.ConfigDirs[0]
}

// DataHome returns the path for the directory where any new configuration
// files should be written.
func (d Dirs) DataHome() string {
	return d.DataDirs[0]
}

// NewConfigPath joins the given path segments to the ConfigHome to produce a
// path where a new configuration file might be written.
func (d Dirs) NewConfigPath(parts ...string) string {
	return filepath.Join(d.ConfigHome(), filepath.Join(parts...))
}

// NewDataPath joins the given path segments to the DataHome to produce a
// path where a new data file might be written.
func (d Dirs) NewDataPath(parts ...string) string {
	return filepath.Join(d.DataHome(), filepath.Join(parts...))
}

// CachePath joins the given path segments to the CacheHome to produce a
// path for a cache file or directory.
func (d Dirs) CachePath(parts ...string) string {
	return filepath.Join(d.CacheDir, filepath.Join(parts...))
}

// ConfigSearchPaths joins the given path segments to each of the directories
// in in ConfigDirs to produce a more specific set of paths to be searched
// in preference order.
func (d Dirs) ConfigSearchPaths(parts ...string) []string {
	return searchPaths(d.ConfigDirs, parts...)
}

// DataSearchPaths joins the given path segments to each of the directories
// in in ConfigDirs to produce a more specific set of paths to be searched
// in preference order.
func (d Dirs) DataSearchPaths(parts ...string) []string {
	return searchPaths(d.DataDirs, parts...)
}

// FindConfigFiles scans over all of the paths in ConfigDirs and tests whether
// a file of the given name is present in each, returning a slice of full
// paths that matched.
func (d Dirs) FindConfigFiles(parts ...string) []string {
	return findFiles(d.ConfigDirs, parts...)
}

// FindDataFiles scans over all of the paths in ConfigDirs and tests whether
// a file of the given name is present in each, returning a slice of full
// paths that matched.
func (d Dirs) FindDataFiles(parts ...string) []string {
	return findFiles(d.DataDirs, parts...)
}

// GlobConfigFiles joins the given parts to create a glob pattern and then
// applies it relative to each of the paths in ConfigDirs, returning all
// of the matches in a single slice.
//
// The order of the result preserves the directory preference order and
// sorts multiple files within the same directory lexicographically.
//
// Remember that on some platforms the config dirs and data dirs overlap,
// so to be robust you should use distinct naming patterns for configuration
// and data files to avoid accidentally matching data files with this method.
func (d Dirs) GlobConfigFiles(parts ...string) []string {
	return globFiles(d.ConfigDirs, parts...)
}

// GlobDataFiles joins the given parts to create a glob pattern and then
// applies it relative to each of the paths in DataDirs, returning all
// of the matches in a single slice.
//
// The order of the result preserves the directory preference order and
// sorts multiple files within the same directory lexicographically.
//
// Remember that on some platforms the config dirs and data dirs overlap,
// so to be robust you should use distinct naming patterns for configuration
// and data files to avoid accidentally matching configuration files with this
// method.
func (d Dirs) GlobDataFiles(parts ...string) []string {
	return globFiles(d.DataDirs, parts...)
}

func searchPaths(bases []string, parts ...string) []string {
	extra := filepath.Join(parts...)
	ret := make([]string, len(bases))
	for i, base := range bases {
		ret[i] = filepath.Join(base, extra)
	}
	return ret
}

func findFiles(bases []string, parts ...string) []string {
	extra := filepath.Join(parts...)
	ret := make([]string, 0, len(bases))
	for _, base := range bases {
		candidate := filepath.Join(base, extra)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			ret = append(ret, candidate)
		}
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func globFiles(bases []string, parts ...string) []string {
	extra := filepath.Join(parts...)
	var ret []string
	for _, base := range bases {
		pattern := filepath.Join(base, extra)
		found, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		ret = append(ret, found...)
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
