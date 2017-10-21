package module

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// moduleManifest is the serialization structure used to record the stored
// module's metadata.
type moduleManifest struct {
	Modules []moduleRecord
}

// moduleRecords represents the stored module's metadata.
// This is compared for equality using '==', so all fields needs to remain
// comparable.
type moduleRecord struct {
	// Source is the module source string, minus any subdirectory.
	// If it is sourced from a registry, it will include the hostname if it is
	// supplied in configuration.
	Source string

	// Version is the exact version string that is stored in this Key.
	Version string

	// Dir is the directory name returned by the FileStorage. This is what
	// allows us to correlate a particular module version with the location on
	// disk.
	Dir string

	// Root is the root directory containing the module. If the module is
	// unpacked from an archive, and not located in the root directory, this is
	// used to direct the loader to the correct subdirectory. This is
	// independent from any subdirectory in the original source string, which
	// may traverse further into the module tree.
	Root string
}

// Return the path to the manifest in parent of the storage directory dir.
func moduleManifestPath(dir string) string {
	const filename = "modules.json"
	// Get the parent directory.
	// The current FolderStorage implementation needed to be able to create
	// this directory, so we can be reasonably certain we can use it.
	parent := filepath.Dir(filepath.Clean(dir))
	return filepath.Join(parent, filename)
}

// loadManifest returns the moduleManifest file from the parent directory.
func loadManifest(dir string) (moduleManifest, error) {
	manifest := moduleManifest{}

	manifestPath := moduleManifestPath(dir)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return manifest, err
	}

	if len(data) == 0 {
		return manifest, nil
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// Store the location of the module, along with the version used and the module
// root directory. The storage method loads the entire file and rewrites it
// each time. This is only done a few times during init, so efficiency is
// not a concern.
func recordModule(dir string, m moduleRecord) error {
	manifest, err := loadManifest(dir)
	if err != nil {
		// if there was a problem with the file, we will attempt to write a new
		// one. Any non-data related error should surface there.
		log.Printf("[WARN] error reading module manifest from %q: %s", dir, err)
	}

	// do nothing if we already have the exact module
	for i, stored := range manifest.Modules {
		if m == stored {
			return nil
		}

		// they are not equal, but if the storage path is the same we need to
		// remove this record to be replaced.
		if m.Dir == stored.Dir {
			manifest.Modules[i] = manifest.Modules[len(manifest.Modules)-1]
			manifest.Modules = manifest.Modules[:len(manifest.Modules)-1]
			break
		}
	}

	manifest.Modules = append(manifest.Modules, m)

	js, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}

	manifestPath := moduleManifestPath(dir)
	return ioutil.WriteFile(manifestPath, js, 0644)
}

// return only the root directory of the module stored in dir.
func getModuleRoot(dir string) (string, error) {
	manifest, err := loadManifest(dir)
	if err != nil {
		return "", err
	}

	for _, m := range manifest.Modules {
		if m.Dir == dir {
			return m.Root, nil
		}
	}
	return "", nil
}

// record only the Root directory for the module stored at dir.
// TODO: remove this compatibility function to store the full moduleRecord.
func recordModuleRoot(dir, root string) error {
	m := moduleRecord{
		Dir:  dir,
		Root: root,
	}

	return recordModule(dir, m)
}
