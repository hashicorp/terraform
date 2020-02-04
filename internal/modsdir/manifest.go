package modsdir

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/addrs"
)

// Record represents some metadata about an installed module, as part
// of a ModuleManifest.
type Record struct {
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
	VersionStr string `json:"Version,omitempty"`

	// Dir is the path to the local directory where the module is installed.
	Dir string `json:"Dir"`
}

// Manifest is a map used to keep track of the filesystem locations
// and other metadata about installed modules.
//
// The configuration loader refers to this, while the module installer updates
// it to reflect any changes to the installed modules.
type Manifest map[string]Record

func (m Manifest) ModuleKey(path addrs.Module) string {
	return path.String()
}

// manifestSnapshotFile is an internal struct used only to assist in our JSON
// serialization of manifest snapshots. It should not be used for any other
// purpose.
type manifestSnapshotFile struct {
	Records []Record `json:"Modules"`
}

func ReadManifestSnapshot(r io.Reader) (Manifest, error) {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(src) == 0 {
		// This should never happen, but we'll tolerate it as if it were
		// a valid empty JSON object.
		return make(Manifest), nil
	}

	var read manifestSnapshotFile
	err = json.Unmarshal(src, &read)

	new := make(Manifest)
	for _, record := range read.Records {
		if record.VersionStr != "" {
			record.Version, err = version.NewVersion(record.VersionStr)
			if err != nil {
				return nil, fmt.Errorf("invalid version %q for %s: %s", record.VersionStr, record.Key, err)
			}
		}

		// Ensure Windows is using the proper modules path format after
		// reading the modules manifest Dir records
		record.Dir = filepath.FromSlash(record.Dir)

		if _, exists := new[record.Key]; exists {
			// This should never happen in any valid file, so we'll catch it
			// and report it to avoid confusing/undefined behavior if the
			// snapshot file was edited incorrectly outside of Terraform.
			return nil, fmt.Errorf("snapshot file contains two records for path %s", record.Key)
		}
		new[record.Key] = record
	}
	return new, nil
}

func ReadManifestSnapshotForDir(dir string) (Manifest, error) {
	fn := filepath.Join(dir, ManifestSnapshotFilename)
	r, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return make(Manifest), nil // missing file is okay and treated as empty
		}
		return nil, err
	}
	return ReadManifestSnapshot(r)
}

func (m Manifest) WriteSnapshot(w io.Writer) error {
	var write manifestSnapshotFile

	for _, record := range m {
		// Make sure VersionStr is in sync with Version, since we encourage
		// callers to manipulate Version and ignore VersionStr.
		if record.Version != nil {
			record.VersionStr = record.Version.String()
		} else {
			record.VersionStr = ""
		}

		// Ensure Dir is written in a format that can be read by Linux and
		// Windows nodes for remote and apply compatibility
		record.Dir = filepath.ToSlash(record.Dir)
		write.Records = append(write.Records, record)
	}

	src, err := json.Marshal(write)
	if err != nil {
		return err
	}

	_, err = w.Write(src)
	return err
}

func (m Manifest) WriteSnapshotToDir(dir string) error {
	fn := filepath.Join(dir, ManifestSnapshotFilename)
	log.Printf("[TRACE] modsdir: writing modules manifest to %s", fn)
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	return m.WriteSnapshot(w)
}
