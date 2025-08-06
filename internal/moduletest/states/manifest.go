package states

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type StateReason string

const (
	StateReasonNone  StateReason = ""
	StateReasonSkip  StateReason = "skip_cleanup"
	StateReasonDep   StateReason = "dependency"
	StateReasonError StateReason = "error"
)

// TestManifest represents the structure of the manifest file that keeps track
// of the state files left-over during test runs.
type TestManifest struct {
	Version int                          `json:"version"`
	Files   map[string]*TestFileManifest `json:"files"`

	dataDir string // Directory where all test-related data is stored
	ids     map[string]bool
}

// TestFileManifest represents a single file with its states keyed by the state
// key.
type TestFileManifest struct {
	States map[string]*TestRunManifest `json:"states"` // Map of state keys to their manifests.
}

// TestRunManifest represents an individual test run state.
type TestRunManifest struct {
	// ID of the state file, used for identification. This will be empty if the
	// state was written to a real backend and not stored locally.
	ID string `json:"id,omitempty"`

	// Reason for the state being left over
	Reason StateReason `json:"reason,omitempty"`
}

// LoadManifest loads the test manifest from the specified root directory.
func LoadManifest(rootDir string) (*TestManifest, error) {
	wd := workdir.NewDir(rootDir)

	manifest := &TestManifest{
		Version: 0,
		Files:   make(map[string]*TestFileManifest),
		dataDir: wd.TestDataDir(),
		ids:     make(map[string]bool),
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

	for _, fileManifest := range manifest.Files {
		for _, runManifest := range fileManifest.States {
			// keep a cache of all known ids
			manifest.ids[runManifest.ID] = true
		}
	}

	return manifest, nil
}

// Save saves the current state of the manifest to the data directory.
func (manifest *TestManifest) Save() error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(manifest.filePath(), data, 0644)
}

// LoadStates loads the states for the specified file.
func (manifest *TestManifest) LoadStates(file *moduletest.File, factory func(string) backend.InitFn) (map[string]*TestRunState, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	allStates := make(map[string]*TestRunState)

	var existingStates map[string]*TestRunManifest
	if fm, exists := manifest.Files[file.Name]; exists {
		existingStates = fm.States
	}

	for _, run := range file.Runs {
		key := run.Config.StateKey
		if existing, exists := allStates[key]; exists {

			if run.Config.Backend != nil {
				f := factory(run.Config.Backend.Type)
				if f == nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unknown backend type",
						Detail:   fmt.Sprintf("Backend type %q is not a recognised backend.", run.Config.Backend.Type),
						Subject:  run.Config.Backend.DeclRange.Ptr(),
					})
					continue
				}

				be, err := getBackendInstance(run.Config.StateKey, run.Config.Backend, f)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid backend configuration",
						Detail:   fmt.Sprintf("Backend configuration was invalid: %s.", err),
						Subject:  run.Config.Backend.DeclRange.Ptr(),
					})
					continue
				}

				// Save the backend for this state when we find it, even if the
				// state was initialised first.
				existing.Backend = be
			}

			continue
		}

		var backend backend.Backend
		if run.Config.Backend != nil {
			// Then we have to load the state from the backend instead of
			// locally or creating a new one.

			f := factory(run.Config.Backend.Type)
			if f == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unknown backend type",
					Detail:   fmt.Sprintf("Backend type %q is not a recognised backend.", run.Config.Backend.Type),
					Subject:  run.Config.Backend.DeclRange.Ptr(),
				})
				continue
			}

			be, err := getBackendInstance(run.Config.StateKey, run.Config.Backend, f)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid backend configuration",
					Detail:   fmt.Sprintf("Backend configuration was invalid: %s.", err),
					Subject:  run.Config.Backend.DeclRange.Ptr(),
				})
				continue
			}

			backend = be
		}

		if existing := existingStates[key]; existing != nil {

			var state *states.State
			if len(existing.ID) > 0 {
				s, err := manifest.loadState(existing)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to load state",
						fmt.Sprintf("Failed to load state from manifest file for %s: %s", run.Name, err)))
					continue
				}
				state = s
			} else {
				state = states.NewState()
			}

			allStates[key] = &TestRunState{
				Run: run,
				Manifest: &TestRunManifest{ // copy this, so we can edit without affecting the original
					ID:     existing.ID,
					Reason: existing.Reason,
				},
				State:   state,
				Backend: backend,
			}
		} else {
			var id string
			if backend == nil {
				id = manifest.generateID()
			}

			allStates[key] = &TestRunState{
				Run: run,
				Manifest: &TestRunManifest{
					ID:     id,
					Reason: StateReasonNone,
				},
				State:   states.NewState(),
				Backend: backend,
			}
		}
	}

	for key := range existingStates {
		if _, exists := allStates[key]; !exists {
			stateKey := key
			if stateKey == configs.TestMainStateIdentifier {
				stateKey = "for the module under test"
			}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Orphaned state",
				fmt.Sprintf("The state key %s is stored in the state manifest indicating a failed cleanup operation, but the state key is not claimed by any run blocks within the current test file. Either restore a run block that manages the specified state, or manually cleanup this state file.", stateKey)))
		}
	}

	return allStates, diags
}

func (manifest *TestManifest) loadState(state *TestRunManifest) (*states.State, error) {
	stateFile := statemgr.NewFilesystem(manifest.StateFilePath(state.ID))
	if err := stateFile.RefreshState(); err != nil {
		return nil, fmt.Errorf("error loading state from file %s: %w", manifest.StateFilePath(state.ID), err)
	}
	return stateFile.State(), nil
}

// SaveStates saves the states for the specified file to the manifest.
func (manifest *TestManifest) SaveStates(file *moduletest.File, states map[string]*TestRunState) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if existingStates, exists := manifest.Files[file.Name]; exists {

		// If we have existing states, we're doing update or delete operations
		// rather than just adding new states.

		for key, existingState := range existingStates.States {

			// First, check all the existing states against the states being
			// saved.

			if state, exists := states[key]; exists {

				// If we have a new state, then overwrite the existing one
				// assuming that it has a reason to be saved.

				if state.Backend != nil {
					// If we have a backend, regardless of the reason, then
					// we'll save the state to the backend.

					stmgr, err := state.Backend.StateMgr(backend.DefaultStateName)
					if err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Failed to write state",
							Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
						})
						continue
					}

					if err := stmgr.WriteState(state.State); err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Failed to write state",
							Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
						})
						continue
					}

					// But, still keep the manifest file itself up-to-date.

					if state.Manifest.Reason != StateReasonNone {
						existingStates.States[key] = state.Manifest
					} else {
						delete(existingStates.States, key)
					}

				} else if state.Manifest.Reason != StateReasonNone {
					if err := manifest.writeState(state); err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Failed to write state",
							Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
						})
						continue
					}
					existingStates.States[key] = state.Manifest
					continue
				} else {

					// If no reason to be saved, then it means we managed to
					// clean everything up properly. So we'll delete the
					// existing state file and remove any mention of it.

					if err := manifest.deleteState(existingState); err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Failed to delete state",
							Detail:   fmt.Sprintf("Failed to delete state file for key %s: %s.", key, err),
						})
						continue
					}
					delete(existingStates.States, key) // remove the state from the manifest file
				}
			}

			// Otherwise, we just leave the state file as is. We don't want to
			// remove it prematurely, as users might still need it to tidy
			// something up.

		}

		// now, we've updated / removed any pre-existing states we should also
		// write any states that are brand new, and weren't in the existing
		// state.

		for key, state := range states {
			if _, exists := existingStates.States[key]; exists {
				// we've already handled everything in the existing state
				continue
			}

			if state.Backend != nil {

				stmgr, err := state.Backend.StateMgr(backend.DefaultStateName)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}

				if err := stmgr.WriteState(state.State); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}

				if state.Manifest.Reason != StateReasonNone {
					existingStates.States[key] = state.Manifest
				}
			} else if state.Manifest.Reason != StateReasonNone {
				if err := manifest.writeState(state); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}
				existingStates.States[key] = state.Manifest
			}
		}

		if len(existingStates.States) == 0 {
			// if we now have tidied everything up, remove record of this from
			// the manifest.
			delete(manifest.Files, file.Name)
		}

	} else {

		// We're just writing entirely new states, so we can just create a new
		// TestFileManifest and add it to the manifest.

		newStates := make(map[string]*TestRunManifest)
		for key, state := range states {
			if state.Backend != nil {

				stmgr, err := state.Backend.StateMgr(backend.DefaultStateName)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}

				if err := stmgr.WriteState(state.State); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}

				if state.Manifest.Reason != StateReasonNone {
					newStates[key] = state.Manifest
				}
			} else if state.Manifest.Reason != StateReasonNone {
				if err := manifest.writeState(state); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to write state",
						Detail:   fmt.Sprintf("Failed to write state file for key %s: %s.", key, err),
					})
					continue
				}
				newStates[key] = state.Manifest
			}
		}

		if len(newStates) > 0 {

			// only add this into the manifest if we actually wrote any
			// new states

			manifest.Files[file.Name] = &TestFileManifest{
				States: newStates,
			}
		}
	}

	return diags
}

func (manifest *TestManifest) writeState(state *TestRunState) error {
	stateFile := statemgr.NewFilesystem(manifest.StateFilePath(state.Manifest.ID))
	if err := stateFile.WriteState(state.State); err != nil {
		return fmt.Errorf("error writing state to file %s: %w", manifest.StateFilePath(state.Manifest.ID), err)
	}
	return nil
}

func (manifest *TestManifest) deleteState(runManifest *TestRunManifest) error {
	target := manifest.StateFilePath(runManifest.ID)
	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, we can ignore this error.
			return nil
		}
		return fmt.Errorf("error deleting state file %s: %w", target, err)
	}
	return nil
}

func (manifest *TestManifest) generateID() string {
	const maxAttempts = 10

	for ix := 0; ix < maxAttempts; ix++ {
		var b [8]byte
		for i := range b {
			n := rand.IntN(len(alphanumeric))
			b[i] = alphanumeric[n]
		}

		id := string(b[:])
		if _, exists := manifest.ids[id]; exists {
			continue // generate another one
		}

		manifest.ids[id] = true
		return id
	}

	panic("failed to generate a unique id 10 times")
}

func (manifest *TestManifest) ensureDataDir() error {
	if _, err := os.Stat(manifest.dataDir); os.IsNotExist(err) {
		return os.MkdirAll(manifest.dataDir, 0755)
	}
	return nil
}

// filePath returns the path to the manifest file
func (manifest *TestManifest) filePath() string {
	return filepath.Join(manifest.dataDir, "manifest.json")
}

// StateFilePath returns the path to the state file for a given ID.
//
// Visible for testing purposes.
func (manifest *TestManifest) StateFilePath(id string) string {
	return filepath.Join(manifest.dataDir, fmt.Sprintf("%s.tfstate", id))
}

// getBackendInstance uses the config for a given run block's backend block to create and return a configured
// instance of that backend type.
func getBackendInstance(stateKey string, config *configs.Backend, f backend.InitFn) (backend.Backend, error) {
	b := f()
	log.Printf("[TRACE] TestConfigTransformer.Transform: instantiated backend of type %T", b)

	schema := b.ConfigSchema()
	decSpec := schema.NoneRequired().DecoderSpec()
	configVal, hclDiags := hcldec.Decode(config.Config, decSpec, nil)
	if hclDiags.HasErrors() {
		return nil, fmt.Errorf("error decoding backend configuration for state key %s : %v", stateKey, hclDiags.Errs())
	}

	if !configVal.IsWhollyKnown() {
		return nil, fmt.Errorf("unknown values within backend definition for state key %s", stateKey)
	}

	newVal, validateDiags := b.PrepareConfig(configVal)
	validateDiags = validateDiags.InConfigBody(config.Config, "")
	if validateDiags.HasErrors() {
		return nil, validateDiags.Err()
	}

	configureDiags := b.Configure(newVal)
	configureDiags = configureDiags.InConfigBody(config.Config, "")
	if validateDiags.HasErrors() {
		return nil, configureDiags.Err()
	}

	return b, nil
}
