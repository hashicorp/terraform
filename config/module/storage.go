package module

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const manifestName = "modules.json"

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

// moduleStorgae implements methods to record and fetch metadata about the
// modules that have been fetched and stored locally. The getter.Storgae
// abstraction doesn't provide the information needed to know which versions of
// a module have been stored, or their location.
type moduleStorage struct {
	storageDir string
}

// loadManifest returns the moduleManifest file from the parent directory.
func (m moduleStorage) loadManifest() (moduleManifest, error) {
	manifest := moduleManifest{}

	manifestPath := filepath.Join(m.storageDir, manifestName)
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
func (m moduleStorage) recordModule(rec moduleRecord) error {
	manifest, err := m.loadManifest()
	if err != nil {
		// if there was a problem with the file, we will attempt to write a new
		// one. Any non-data related error should surface there.
		log.Printf("[WARN] error reading module manifest: %s", err)
	}

	// do nothing if we already have the exact module
	for i, stored := range manifest.Modules {
		if rec == stored {
			return nil
		}

		// they are not equal, but if the storage path is the same we need to
		// remove this rec to be replaced.
		if rec.Dir == stored.Dir {
			manifest.Modules[i] = manifest.Modules[len(manifest.Modules)-1]
			manifest.Modules = manifest.Modules[:len(manifest.Modules)-1]
			break
		}
	}

	manifest.Modules = append(manifest.Modules, rec)

	js, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}

	manifestPath := filepath.Join(m.storageDir, manifestName)
	return ioutil.WriteFile(manifestPath, js, 0644)
}

// return only the root directory of the module stored in dir.
func (m moduleStorage) getModuleRoot(dir string) (string, error) {
	manifest, err := m.loadManifest()
	if err != nil {
		return "", err
	}

	for _, mod := range manifest.Modules {
		if mod.Dir == dir {
			return mod.Root, nil
		}
	}
	return "", nil
}

// record only the Root directory for the module stored at dir.
// TODO: remove this compatibility function to store the full moduleRecord.
func (m moduleStorage) recordModuleRoot(dir, root string) error {
	rec := moduleRecord{
		Dir:  dir,
		Root: root,
	}

	return m.recordModule(rec)
}
