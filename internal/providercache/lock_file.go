package providercache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// lockFile represents a file on disk that captures selected versions and
// their associated package checksums resulting from an install process, so
// that later consumers of that install process can be sure they are reading
// an identical set of providers to what the install process intended.
//
// This is an internal type used to encapsulate the reading, parsing,
// serializing, and writing of lock files. Its public interface is via methods
// on type Installer.
type lockFile struct {
	filename string
}

// LockFileEntry represents an entry for a specific provider in a LockFile.
type lockFileEntry struct {
	SelectedVersion getproviders.Version
	PackageHash     string
}

var _ json.Marshaler = (*lockFileEntry)(nil)
var _ json.Unmarshaler = (*lockFileEntry)(nil)

// Read returns the current locks captured in the lock file.
//
// If the file does not exist, the result is successful but empty to indicate
// that no providers at all are available for use.
func (lf *lockFile) Read() (map[addrs.Provider]lockFileEntry, error) {
	buf, err := ioutil.ReadFile(lf.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no file means no locks yet
		}
		return nil, err
	}

	var rawEntries map[string]*lockFileEntry
	err = json.Unmarshal(buf, &rawEntries)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %s", lf.filename, err)
	}

	ret := make(map[addrs.Provider]lockFileEntry, len(rawEntries))
	for providerStr, entry := range rawEntries {
		provider, diags := addrs.ParseProviderSourceString(providerStr)
		if diags.HasErrors() {
			// This file is both generated and consumed by Terraform, so we
			// don't use super-detailed error messages for problems in it.
			// If we get here without someone tampering with the file then
			// it's presumably a bug in either our serializer or our parser.
			return nil, fmt.Errorf("error parsing %s: invalid provider address %q", lf.filename, providerStr)
		}
		ret[provider] = *entry
	}

	return ret, nil
}

// Write stores a new set of entries in the lock file, disarding any
// selections previously stored there.
func (lf *lockFile) Write(new map[addrs.Provider]lockFileEntry) error {
	toStore := make(map[string]*lockFileEntry, len(new))
	for provider := range new {
		entry := new[provider] // so that each reference below is to a different object
		toStore[provider.String()] = &entry
	}

	buf, err := json.MarshalIndent(toStore, "", "  ")
	if err != nil {
		return fmt.Errorf("error writing %s: %s", lf.filename, err)
	}

	os.MkdirAll(
		filepath.Dir(lf.filename), 0775,
	) // ignore error since WriteFile below will generate a better one anyway
	return ioutil.WriteFile(lf.filename, buf, 0664)
}

func (lfe *lockFileEntry) UnmarshalJSON(src []byte) error {
	type Raw struct {
		VersionStr string `json:"version"`
		Hash       string `json:"hash"`
	}
	var raw Raw
	err := json.Unmarshal(src, &raw)
	if err != nil {
		return err
	}
	version, err := getproviders.ParseVersion(raw.VersionStr)
	if err != nil {
		return fmt.Errorf("invalid version number: %s", err)
	}
	lfe.SelectedVersion = version
	lfe.PackageHash = raw.Hash
	return nil
}

func (lfe *lockFileEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"version": lfe.SelectedVersion.String(),
		"hash":    lfe.PackageHash,
	})
}
