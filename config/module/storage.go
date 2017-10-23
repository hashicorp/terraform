package module

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"

	getter "github.com/hashicorp/go-getter"
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

// moduleStorage implements methods to record and fetch metadata about the
// modules that have been fetched and stored locally. The getter.Storgae
// abstraction doesn't provide the information needed to know which versions of
// a module have been stored, or their location.
type moduleStorage struct {
	getter.Storage
	storageDir string
}

func newModuleStorage(s getter.Storage) moduleStorage {
	return moduleStorage{
		Storage:    s,
		storageDir: storageDir(s),
	}
}

// The Tree needs to know where to store the module manifest.
// Th Storage abstraction doesn't provide access to the storage root directory,
// so we extract it here.
// TODO: This needs to be replaced by refactoring the getter.Storage usage for
//       modules.
func storageDir(s getter.Storage) string {
	// get the StorageDir directly if possible
	switch t := s.(type) {
	case *getter.FolderStorage:
		return t.StorageDir
	case moduleStorage:
		return t.storageDir
	}

	// this should be our UI wrapper which is exported here, so we need to
	// extract the FolderStorage via reflection.
	fs := reflect.ValueOf(s).Elem().FieldByName("Storage").Interface()
	return storageDir(fs.(getter.Storage))
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

func (m moduleStorage) getStorage(key string, src string, mode GetMode) (string, bool, error) {
	// Get the module with the level specified if we were told to.
	if mode > GetModeNone {
		log.Printf("[DEBUG] fetching %q with key %q", src, key)
		if err := m.Storage.Get(key, src, mode == GetModeUpdate); err != nil {
			return "", false, err
		}
	}

	// Get the directory where the module is.
	dir, found, err := m.Storage.Dir(key)
	log.Printf("[DEBUG] found %q in %q: %t", src, dir, found)
	return dir, found, err
}
