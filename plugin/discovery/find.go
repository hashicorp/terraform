package discovery

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

const machineName = runtime.GOOS + "_" + runtime.GOARCH

// FindPlugins looks in the given directories for files whose filenames
// suggest that they are plugins of the given kind (e.g. "provider") and
// returns a PluginMetaSet representing the discovered potential-plugins.
//
// Currently this supports two different naming schemes. The current
// standard naming scheme is a subdirectory called $GOOS-$GOARCH containing
// files named terraform-$KIND-$NAME-V$VERSION. The legacy naming scheme is
// files directly in the given directory whose names are like
// terraform-$KIND-$NAME.
//
// Only one plugin will be returned for each unique plugin (name, version)
// pair, with preference given to files found in earlier directories.
//
// This is a convenience wrapper around FindPluginPaths and ResolvePluginsPaths.
func FindPlugins(kind string, dirs []string) PluginMetaSet {
	return ResolvePluginPaths(FindPluginPaths(kind, dirs))
}

// FindPluginPaths looks in the given directories for files whose filenames
// suggest that they are plugins of the given kind (e.g. "provider").
//
// The return value is a list of absolute paths that appear to refer to
// plugins in the given directories, based only on what can be inferred
// from the naming scheme. The paths returned are ordered such that files
// in later dirs appear after files in earlier dirs in the given directory
// list. Within the same directory plugins are returned in a consistent but
// undefined order.
func FindPluginPaths(kind string, dirs []string) []string {
	// This is just a thin wrapper around findPluginPaths so that we can
	// use the latter in tests with a fake machineName so we can use our
	// test fixtures.
	return findPluginPaths(kind, machineName, dirs)
}

func findPluginPaths(kind string, machineName string, dirs []string) []string {
	prefix := "terraform-" + kind + "-"

	ret := make([]string, 0, len(dirs))

	for _, baseDir := range dirs {
		baseItems, err := ioutil.ReadDir(baseDir)
		if err != nil {
			// Ignore missing dirs, non-dirs, etc
			continue
		}

		for _, item := range baseItems {
			fullName := item.Name()

			if fullName == machineName && item.Mode().IsDir() {
				// Current-style $GOOS-$GOARCH directory prefix
				machineDir := filepath.Join(baseDir, machineName)
				machineItems, err := ioutil.ReadDir(machineDir)
				if err != nil {
					continue
				}

				for _, item := range machineItems {
					fullName := item.Name()

					if !strings.HasPrefix(fullName, prefix) {
						continue
					}

					// New-style paths must have a version segment in filename
					if !strings.Contains(fullName, "-V") {
						continue
					}

					absPath, err := filepath.Abs(filepath.Join(machineDir, fullName))
					if err != nil {
						continue
					}

					ret = append(ret, filepath.Clean(absPath))
				}

				continue
			}

			if strings.HasPrefix(fullName, prefix) {
				// Legacy style with files directly in the base directory
				absPath, err := filepath.Abs(filepath.Join(baseDir, fullName))
				if err != nil {
					continue
				}

				ret = append(ret, filepath.Clean(absPath))
			}
		}
	}

	return ret
}

// ResolvePluginPaths takes a list of paths to plugin executables (as returned
// by e.g. FindPluginPaths) and produces a PluginMetaSet describing the
// referenced plugins.
//
// If the same combination of plugin name and version appears multiple times,
// the earlier reference will be preferred. Several different versions of
// the same plugin name may be returned, in which case the methods of
// PluginMetaSet can be used to filter down.
func ResolvePluginPaths(paths []string) PluginMetaSet {
	s := make(PluginMetaSet)

	type nameVersion struct {
		Name    string
		Version string
	}
	found := make(map[nameVersion]struct{})

	for _, path := range paths {
		baseName := filepath.Base(path)
		if !strings.HasPrefix(baseName, "terraform-") {
			// Should never happen with reasonable input
			continue
		}
		baseName = baseName[10:]
		firstDash := strings.Index(baseName, "-")
		if firstDash == -1 {
			// Should never happen with reasonable input
			continue
		}

		baseName = baseName[firstDash+1:]
		if baseName == "" {
			// Should never happen with reasonable input
			continue
		}

		parts := strings.SplitN(baseName, "-V", 2)
		name := parts[0]
		version := "0.0.0"
		if len(parts) == 2 {
			version = parts[1]
		}

		if _, ok := found[nameVersion{name, version}]; ok {
			// Skip duplicate versions of the same plugin
			// (We do this during this step because after this we will be
			// dealing with sets and thus lose our ordering with which to
			// decide preference.)
			continue
		}

		s.Add(PluginMeta{
			Name:    name,
			Version: VersionStr(version),
			Path:    path,
		})
		found[nameVersion{name, version}] = struct{}{}
	}

	return s
}
