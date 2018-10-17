package planfile

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strings"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/configs/configload"
)

const configSnapshotPrefix = "tfconfig/"
const configSnapshotManifestFile = configSnapshotPrefix + "modules.json"
const configSnapshotModulePrefix = configSnapshotPrefix + "m-"

type configSnapshotModuleRecord struct {
	Key        string `json:"Key"`
	SourceAddr string `json:"Source,omitempty"`
	VersionStr string `json:"Version,omitempty"`
	Dir        string `json:"Dir"`
}
type configSnapshotModuleManifest []configSnapshotModuleRecord

func readConfigSnapshot(z *zip.Reader) (*configload.Snapshot, error) {
	// Errors from this function are expected to be reported with some
	// additional prefix context about them being in a config snapshot,
	// so they should not themselves refer to the config snapshot.
	// They are also generally indicative of an invalid file, and so since
	// plan files should not be hand-constructed we don't need to worry
	// about making the messages user-actionable.

	snap := &configload.Snapshot{
		Modules: map[string]*configload.SnapshotModule{},
	}
	var manifestSrc []byte

	// For processing our source files, we'll just sweep over all the files
	// and react to the one-by-one to start, and then clean up afterwards
	// when we'll presumably have found the manifest file.
	for _, file := range z.File {
		switch {

		case file.Name == configSnapshotManifestFile:
			// It's the manifest file, so we'll just read it raw into
			// manifestSrc for now and process it below.
			r, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open module manifest: %s", r)
			}
			manifestSrc, err = ioutil.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read module manifest: %s", r)
			}

		case strings.HasPrefix(file.Name, configSnapshotModulePrefix):
			relName := file.Name[len(configSnapshotModulePrefix):]
			moduleKey, fileName := path.Split(relName)

			// moduleKey should currently have a trailing slash on it, which we
			// can use to recognize the difference between the root module
			// (just a trailing slash) and no module path at all (empty string).
			if moduleKey == "" {
				// ignore invalid config entry
				continue
			}
			moduleKey = moduleKey[:len(moduleKey)-1] // trim trailing slash

			r, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open snapshot of %s from module %q: %s", fileName, moduleKey, err)
			}
			fileSrc, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read snapshot of %s from module %q: %s", fileName, moduleKey, err)
			}

			if _, exists := snap.Modules[moduleKey]; !exists {
				snap.Modules[moduleKey] = &configload.SnapshotModule{
					Files: map[string][]byte{},
					// Will fill in everything else afterwards, when we
					// process the manifest.
				}
			}
			snap.Modules[moduleKey].Files[fileName] = fileSrc
		}
	}

	if manifestSrc == nil {
		return nil, fmt.Errorf("config snapshot does not have manifest file")
	}

	var manifest configSnapshotModuleManifest
	err := json.Unmarshal(manifestSrc, &manifest)
	if err != nil {
		return nil, fmt.Errorf("invalid module manifest: %s", err)
	}

	for _, record := range manifest {
		modSnap, exists := snap.Modules[record.Key]
		if !exists {
			// We'll allow this, assuming that it's a module with no files.
			// This is still weird, since we generally reject modules with
			// no files, but we'll allow it because downstream errors will
			// catch it in that case.
			modSnap = &configload.SnapshotModule{
				Files: map[string][]byte{},
			}
			snap.Modules[record.Key] = modSnap
		}
		modSnap.SourceAddr = record.SourceAddr
		modSnap.Dir = record.Dir
		if record.VersionStr != "" {
			v, err := version.NewVersion(record.VersionStr)
			if err != nil {
				return nil, fmt.Errorf("manifest has invalid version string %q for module %q", record.VersionStr, record.Key)
			}
			modSnap.Version = v
		}
	}

	// Finally, we'll make sure we don't have any errant files for modules that
	// aren't in the manifest.
	for k := range snap.Modules {
		found := false
		for _, record := range manifest {
			if record.Key == k {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("found files for module %q that isn't recorded in the manifest", k)
		}
	}

	return snap, nil
}

// writeConfigSnapshot adds to the given zip.Writer one or more files
// representing the given snapshot.
//
// This file creates new files in the writer, so any already-open writer
// for the file will be invalidated by this call. The writer remains open
// when this function returns.
func writeConfigSnapshot(snap *configload.Snapshot, z *zip.Writer) error {
	// Errors from this function are expected to be reported with some
	// additional prefix context about them being in a config snapshot,
	// so they should not themselves refer to the config snapshot.
	// They are also indicative of a bug in the caller, so they do not
	// need to be user-actionable.

	var manifest configSnapshotModuleManifest
	keys := make([]string, 0, len(snap.Modules))
	for k := range snap.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// We'll re-use this fileheader for each Create we do below.

	for _, k := range keys {
		snapMod := snap.Modules[k]
		record := configSnapshotModuleRecord{
			Dir:        snapMod.Dir,
			Key:        k,
			SourceAddr: snapMod.SourceAddr,
		}
		if snapMod.Version != nil {
			record.VersionStr = snapMod.Version.String()
		}
		manifest = append(manifest, record)

		pathPrefix := fmt.Sprintf("%s%s/", configSnapshotModulePrefix, k)
		for filename, src := range snapMod.Files {
			zh := &zip.FileHeader{
				Name:     pathPrefix + filename,
				Method:   zip.Deflate,
				Modified: time.Now(),
			}
			w, err := z.CreateHeader(zh)
			if err != nil {
				return fmt.Errorf("failed to create snapshot of %s from module %q: %s", zh.Name, k, err)
			}
			_, err = w.Write(src)
			if err != nil {
				return fmt.Errorf("failed to write snapshot of %s from module %q: %s", zh.Name, k, err)
			}
		}
	}

	// Now we'll write our manifest
	{
		zh := &zip.FileHeader{
			Name:     configSnapshotManifestFile,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		src, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize module manifest: %s", err)
		}
		w, err := z.CreateHeader(zh)
		if err != nil {
			return fmt.Errorf("failed to create module manifest: %s", err)
		}
		_, err = w.Write(src)
		if err != nil {
			return fmt.Errorf("failed to write module manifest: %s", err)
		}
	}

	return nil
}
