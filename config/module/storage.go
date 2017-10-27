package module

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	getter "github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/mitchellh/cli"
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
	// Source is the module source string from the config, minus any
	// subdirectory.
	Source string

	// Key is the locally unique identifier for this module.
	Key string

	// Version is the exact version string for the stored module.
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

	// url is the location of the module source
	url string

	// Registry is true if this module is sourced from a registry
	registry bool
}

// ModuleStorage implements methods to record and fetch metadata about the
// modules that have been fetched and stored locally. The getter.Storgae
// abstraction doesn't provide the information needed to know which versions of
// a module have been stored, or their location.
type ModuleStorage struct {
	StorageDir string
	Services   *disco.Disco
	Ui         cli.Ui
	Mode       GetMode
}

// loadManifest returns the moduleManifest file from the parent directory.
func (m ModuleStorage) loadManifest() (moduleManifest, error) {
	manifest := moduleManifest{}

	manifestPath := filepath.Join(m.StorageDir, manifestName)
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
func (m ModuleStorage) recordModule(rec moduleRecord) error {
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

	manifestPath := filepath.Join(m.StorageDir, manifestName)
	return ioutil.WriteFile(manifestPath, js, 0644)
}

// load the manifest from dir, and return all module versions matching the
// provided source. Records with no version info will be skipped, as they need
// to be uniquely identified by other means.
func (m ModuleStorage) moduleVersions(source string) ([]moduleRecord, error) {
	manifest, err := m.loadManifest()
	if err != nil {
		return manifest.Modules, err
	}

	var matching []moduleRecord

	for _, m := range manifest.Modules {
		if m.Source == source && m.Version != "" {
			matching = append(matching, m)
		}
	}

	return matching, nil
}

func (m ModuleStorage) moduleDir(key string) (string, error) {
	manifest, err := m.loadManifest()
	if err != nil {
		return "", err
	}

	for _, m := range manifest.Modules {
		if m.Key == key {
			return m.Dir, nil
		}
	}

	return "", nil
}

// return only the root directory of the module stored in dir.
func (m ModuleStorage) getModuleRoot(dir string) (string, error) {
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
func (m ModuleStorage) recordModuleRoot(dir, root string) error {
	rec := moduleRecord{
		Dir:  dir,
		Root: root,
	}

	return m.recordModule(rec)
}

func (m ModuleStorage) getStorage(key string, src string) (string, bool, error) {
	storage := &getter.FolderStorage{
		StorageDir: m.StorageDir,
	}

	// Get the module with the level specified if we were told to.
	if m.Mode > GetModeNone {
		log.Printf("[DEBUG] fetching %q with key %q", src, key)
		if err := storage.Get(key, src, m.Mode == GetModeUpdate); err != nil {
			return "", false, err
		}
	}

	// Get the directory where the module is.
	dir, found, err := storage.Dir(key)
	log.Printf("[DEBUG] found %q in %q: %t", src, dir, found)
	return dir, found, err
}

// find a stored module that's not from a registry
func (m ModuleStorage) findModule(key string) (string, error) {
	if m.Mode == GetModeUpdate {
		return "", nil
	}

	return m.moduleDir(key)
}

// find a registry module
func (m ModuleStorage) findRegistryModule(mSource, constraint string) (moduleRecord, error) {
	rec := moduleRecord{
		Source: mSource,
	}
	// detect if we have a registry source
	mod, err := regsrc.ParseModuleSource(mSource)
	switch err {
	case nil:
		//ok
	case regsrc.ErrInvalidModuleSource:
		return rec, nil
	default:
		return rec, err
	}
	rec.registry = true

	log.Printf("[TRACE] %q is a registry module", mod.Module())

	versions, err := m.moduleVersions(mod.String())
	if err != nil {
		log.Printf("[ERROR] error looking up versions for %q: %s", mod.Module(), err)
		return rec, err
	}

	match, err := newestRecord(versions, constraint)
	if err != nil {
		// TODO: does this allow previously unversioned modules?
		log.Printf("[INFO] no matching version for %q<%s>, %s", mod.Module(), constraint, err)
	}

	rec.Dir = match.Dir
	rec.Version = match.Version
	found := rec.Dir != ""

	// we need to lookup available versions
	// Only on Get if it's not found, on unconditionally on Update
	if (m.Mode == GetModeGet && !found) || (m.Mode == GetModeUpdate) {
		resp, err := lookupModuleVersions(nil, mod)
		if err != nil {
			return rec, err
		}

		if len(resp.Modules) == 0 {
			return rec, fmt.Errorf("module %q not found in registry", mod.Module())
		}

		match, err := newestVersion(resp.Modules[0].Versions, constraint)
		if err != nil {
			return rec, err
		}

		if match == nil {
			return rec, fmt.Errorf("no versions for %q found matching %q", mod.Module(), constraint)
		}

		rec.Version = match.Version

		rec.url, err = lookupModuleLocation(nil, mod, rec.Version)
		if err != nil {
			return rec, err
		}
	}
	return rec, nil
}
