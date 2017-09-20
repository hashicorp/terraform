package discovery

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

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
	return findPluginPaths(kind, dirs)
}

func findPluginPaths(kind string, dirs []string) []string {
	prefix := "terraform-" + kind + "-"

	ret := make([]string, 0, len(dirs))

	for _, dir := range dirs {
		items, err := ioutil.ReadDir(dir)
		if err != nil {
			// Ignore missing dirs, non-dirs, etc
			continue
		}

		log.Printf("[DEBUG] checking for %s in %q", kind, dir)

		for _, item := range items {
			fullName := item.Name()

			if !strings.HasPrefix(fullName, prefix) {
				continue
			}

			// New-style paths must have a version segment in filename
			if strings.Contains(strings.ToLower(fullName), "_v") {
				absPath, err := filepath.Abs(filepath.Join(dir, fullName))
				if err != nil {
					log.Printf("[ERROR] plugin filepath error: %s", err)
					continue
				}

				log.Printf("[DEBUG] found %s %q", kind, fullName)
				ret = append(ret, filepath.Clean(absPath))
				continue
			}

			// Legacy style with files directly in the base directory
			absPath, err := filepath.Abs(filepath.Join(dir, fullName))
			if err != nil {
				log.Printf("[ERROR] plugin filepath error: %s", err)
				continue
			}

			log.Printf("[WARNING] found legacy %s %q", kind, fullName)

			ret = append(ret, filepath.Clean(absPath))
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
		baseName := strings.ToLower(filepath.Base(path))
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

		// Trim the .exe suffix used on Windows before we start wrangling
		// the remainder of the path.
		if strings.HasSuffix(baseName, ".exe") {
			baseName = baseName[:len(baseName)-4]
		}

		parts := strings.SplitN(baseName, "_v", 2)
		name := parts[0]
		version := VersionZero
		if len(parts) == 2 {
			version = parts[1]
		}

		// Auto-installed plugins contain an extra name portion representing
		// the expected plugin version, which we must trim off.
		if underX := strings.Index(version, "_x"); underX != -1 {
			version = version[:underX]
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
