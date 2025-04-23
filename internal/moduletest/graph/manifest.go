// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type StateReason string

const (
	None        StateReason = ""
	ReasonSkip  StateReason = "skip_cleanup"
	ReasonDep   StateReason = "dependency"
	ReasonError StateReason = "error"
)

// TestManifest represents the structure of the manifest file
// that keeps track of the state files left-over during test runs.
type TestManifest struct {
	Version int                  `json:"version"`
	Files   map[string]*TestFile `json:"files"`

	dataDir string // Directory where all test-related data is stored
}

// TestFile represents a single file with its states keyed by the state key
type TestFile struct {
	States map[string]*TestState `json:"states"`
}

// TestState represents an individual test state
type TestState struct {
	Path   string      `json:"path"`   // Path to the state file
	Reason StateReason `json:"reason"` // Reason for the state being left over
}

// BuildStateManifest creates a manifest for a set of files and their runs.
// The manifest is used to keep track of the state files created during the test runs.
func BuildStateManifest(rootDir string, files map[string]*moduletest.File) (*TestManifest, error) {
	wd := workdir.NewDir(rootDir)
	// Load the manifest or create a new one
	manifest, err := LoadManifest(wd.TestDataDir())
	if err != nil {
		return nil, err
	}

	ids := make(map[string]struct{})
	for _, file := range files {
		manifestFile, exists := manifest.Files[file.Name]
		if !exists {
			manifestFile = &TestFile{States: make(map[string]*TestState)}
		}
		keys := make([]string, 0, len(file.Runs))

		// collect all state keys (implicit or explicit)
		for _, run := range file.Runs {
			keys = append(keys, run.Config.StateKey)
		}

		// create a state file path for each state key
		for _, key := range keys {
			// if the state key already exists in the manifest, we use that existing entry
			if _, ok := manifestFile.States[key]; ok {
				continue
			}
			id := manifest.generateID()
			if _, exists := ids[id]; exists {
				panic(fmt.Sprintf("duplicate generated state id %s", id))
			}
			ids[id] = struct{}{}
			path := filepath.Join(manifest.dataDir, fmt.Sprintf("%s.tfstate", id))
			manifestFile.States[key] = &TestState{Path: path}
		}
		manifest.Files[file.Name] = manifestFile
	}

	// write manifest to disk
	return manifest, manifest.WriteManifest()
}

// LoadManifest loads a manifest from disk, or creates a new one if it doesn't exist
func LoadManifest(dataDir string) (*TestManifest, error) {
	manifest := &TestManifest{
		Version: 0,
		Files:   make(map[string]*TestFile),
		dataDir: dataDir,
	}

	// Create directory if it doesn't exist
	if err := manifest.ensureDataDir(); err != nil {
		return nil, err
	}

	data, err := os.OpenFile(manifest.filePath(), os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer data.Close()

	if err := json.NewDecoder(data).Decode(manifest); err != nil && err != io.EOF {
		return nil, err
	}

	return manifest, nil
}

func (m *TestManifest) Empty() (bool, error) {
	for _, file := range m.Files {
		for _, state := range file.States {
			stateFile := statemgr.NewFilesystem(state.Path)
			if err := stateFile.RefreshState(); err != nil {
				return false, err
			}
			if !stateFile.State().Empty() {
				return false, nil
			}
		}
	}

	return true, nil
}

// writeState writes a state to disk, with the path being the location in the manifest
// where the state is expected to be stored for a given key.
func (m *TestManifest) writeState(key string, state *TestFileState) error {
	// retrieve the path where the manifest expect the state to be stored
	// for this key.
	filename := state.File.Name
	file, exists := m.Files[filename]
	if !exists {
		return fmt.Errorf("file %s not found in manifest", filename)
	}
	location, exists := file.States[key]
	if !exists {
		return fmt.Errorf("state %s not found in file %s", key, filename)
	}
	location.Reason = state.Reason

	// Write state to disk
	stateFile := statemgr.NewFilesystem(location.Path)
	err := stateFile.WriteState(state.State)
	if err != nil {
		return err
	}

	return nil
}

func (m *TestManifest) readState(filename, stateKey string) (*TestFileState, error) {
	file, exists := m.Files[filename]
	if !exists {
		return nil, fmt.Errorf("file %s not found in manifest", filename)
	}
	location, exists := file.States[stateKey]
	if !exists {
		return nil, fmt.Errorf("state %s not found in file %s", stateKey, filename)
	}

	// Read state from disk
	stateFile := statemgr.NewFilesystem(location.Path)
	if err := stateFile.RefreshState(); err != nil {
		return nil, err
	}

	return &TestFileState{State: stateFile.State(), Reason: location.Reason}, nil
}

// WriteManifest writes the manifest to disk
func (m *TestManifest) WriteManifest() error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath(), data, 0644)
}

func (m *TestManifest) generateID() string {
	var b [8]byte
	for i := range b {
		n := rand.IntN(len(alphanumeric))
		b[i] = alphanumeric[n]
	}
	return string(b[:])
}

func (m *TestManifest) ensureDataDir() error {
	if _, err := os.Stat(m.dataDir); os.IsNotExist(err) {
		return os.MkdirAll(m.dataDir, 0755)
	}
	return nil
}

// filePath returns the path to the manifest file
func (m *TestManifest) filePath() string {
	return filepath.Join(m.dataDir, "manifest.json")
}
