package configload

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

// moduleRecord represents some metadata about an installed module, as part
// of a moduleManifest.
type moduleRecord struct {
	// Key is a unique identifier for this particular module, based on its
	// position within the static module tree.
	Key string `json:"Key"`

	// SourceAddr is the source address given for this module in configuration.
	// This is used only to detect if the source was changed in configuration
	// since the module was last installed, which means that the installer
	// must re-install it.
	SourceAddr string `json:"Source"`

	// Version is the exact version of the module, which results from parsing
	// VersionStr. nil for un-versioned modules.
	Version *version.Version `json:"-"`

	// VersionStr is the version specifier string. This is used only for
	// serialization in snapshots and should not be accessed or updated
	// by any other codepaths; use "Version" instead.
	VersionStr string `json:"Version"`

	// Dir is the path to the local directory where the module is installed.
	Dir string `json:"Dir"`
}

// moduleManifest is a map used to keep track of the filesystem locations
// and other metadata about installed modules.
//
// The configuration loader refers to this, while the module installer updates
// it to reflect any changes to the installed modules.
type moduleManifest map[string]moduleRecord

func manifestKey(path []string) string {
	return strings.Join(path, ".")
}

// manifestSnapshotFile is an internal struct used only to assist in our JSON
// serializtion of manifest snapshots. It should not be used for any other
// purposes.
type manifestSnapshotFile struct {
	Records []moduleRecord `json:"Modules"`
}

const manifestFilename = "modules.json"

func (m *moduleMgr) manifestSnapshotPath() string {
	return filepath.Join(m.Dir, manifestFilename)
}

// readModuleManifestSnapshot loads a manifest snapshot from the filesystem.
func (m *moduleMgr) readModuleManifestSnapshot() error {
	src, err := m.FS.ReadFile(m.manifestSnapshotPath())
	if err != nil {
		if os.IsNotExist(err) {
			// We'll treat a missing file as an empty manifest
			m.manifest = make(moduleManifest)
			return nil
		}
		return err
	}

	if len(src) == 0 {
		// This should never happen, but we'll tolerate it as if it were
		// a valid empty JSON object.
		m.manifest = make(moduleManifest)
		return nil
	}

	var read manifestSnapshotFile
	err = json.Unmarshal(src, &read)

	new := make(moduleManifest)
	for _, record := range read.Records {
		if record.VersionStr != "" {
			record.Version, err = version.NewVersion(record.VersionStr)
			if err != nil {
				return fmt.Errorf("invalid version %q for %s: %s", record.VersionStr, record.Key, err)
			}
		}
		if _, exists := new[record.Key]; exists {
			// This should never happen in any valid file, so we'll catch it
			// and report it to avoid confusing/undefined behavior if the
			// snapshot file was edited incorrectly outside of Terraform.
			return fmt.Errorf("snapshot file contains two records for path %s", record.Key)
		}
		new[record.Key] = record
	}

	m.manifest = new

	return nil
}

// writeModuleManifestSnapshot writes a snapshot of the current manifest
// to the filesystem.
//
// The caller must guarantee no concurrent modifications of the manifest for
// the duration of a call to this function, or the behavior is undefined.
func (m *moduleMgr) writeModuleManifestSnapshot() error {
	var write manifestSnapshotFile

	for _, record := range m.manifest {
		// Make sure VersionStr is in sync with Version, since we encourage
		// callers to manipulate Version and ignore VersionStr.
		record.VersionStr = record.Version.String()
		write.Records = append(write.Records, record)
	}

	src, err := json.Marshal(write)
	if err != nil {
		return err
	}

	return m.FS.WriteFile(m.manifestSnapshotPath(), src, os.ModePerm)
}
