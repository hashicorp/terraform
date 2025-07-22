// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestFileState is a helper struct that just maps a run block to the state that
// was produced by the execution of that run block.
type TestFileState struct {
	File   *moduletest.File
	Run    *moduletest.Run
	State  *states.State
	Reason StateReason

	backend runBackend
}

// runBackend connects the backend instance to the run that
// contains it. This can be used to check whether a given run
// should be able to update the remote state or not.
type runBackend struct {
	instance backend.Backend
	run      *moduletest.Run
}

// TestStateTransformer is a GraphTransformer that initializes the context with
// all the states produced by the test file.
type TestStateTransformer struct {
	*graphOptions
	BackendFactory func(string) backend.InitFn
}

func (t *TestStateTransformer) Transform(g *terraform.Graph) error {
	// This map tracks the state of each run in the file. If multiple runs
	// have the same state key, they will share the same state.
	statesMap := make(map[string]*TestFileState)

	// We iterate through all the file's runs. Whenever we identify a state key that
	// hasn't had an internal state set for it yet, we create it.
	for _, run := range t.File.Runs {
		key := run.Config.StateKey
		if _, exists := statesMap[key]; !exists {

			var state *TestFileState

			bc, stateUsesBackend := t.File.Config.BackendConfigs[key]

			switch {
			case stateUsesBackend && bc.Run.Name == run.Name:
				// This state key has an associated backend, and we're processing
				// the node for the run block that controls the backend via a
				// "backend" block.
				//
				// We proceed and set the state using that backend.
				if t.BackendFactory == nil {
					return fmt.Errorf("error retrieving state for state key %q from backend: nil BackendFactory. This is a bug in Terraform and should be reported.", key)
				}

				f := t.BackendFactory(bc.Backend.Type)
				if f == nil {
					return fmt.Errorf("error retrieving state for state key %q from backend: No init function found for backend type %q. This is a bug in Terraform and should be reported.", key, bc.Backend.Type)
				}
				be, err := getBackendInstance(key, bc.Backend, f)
				if err != nil {
					return err
				}

				stmgr, err := be.StateMgr(backend.DefaultStateName) // We only allow use of the default workspace
				if err != nil {
					return fmt.Errorf("error retrieving state for state key %q from backend: error retrieving state manager: %w", key, err)
				}
				err = stmgr.RefreshState()
				if err != nil {
					return fmt.Errorf("error retrieving state for state key %q from backend: error reading state: %w", key, err)
				}

				log.Printf("[TRACE] TestConfigTransformer.Transform: set initial state for state key %q using backend of type %T declared at %s", key, be, bc.Backend.DeclRange)
				state = &TestFileState{
					File:  t.File,
					Run:   nil,
					State: stmgr.State(),
					backend: runBackend{
						instance: be,
						run:      run, // This is the run containing the backend block
					},
				}

			case stateUsesBackend && bc.Run.Name != run.Name:
				// This state key has an associated backend, but we're processing
				// a run block that doesn't include a "backend" block.
				//
				// In this case, do nothing and continue to the next node.
				// The state for this state key will be set when the for loop processes
				// the run block that controls the given backend via a "backend" block.
				continue

			case !stateUsesBackend:
				log.Printf("[TRACE] TestConfigTransformer.Transform: set initial state for state key %q as empty state", key)
				// If no backend is used, we load the in-memory state from the manifest. We should
				// have already initialized the state in the manifest before we get here.
				var err error
				if state, err = t.StateManifest.readState(t.File.Name, key); err != nil {
					return fmt.Errorf("error retrieving state for state key %q from manifest: %w", key, err)
				}
				state.File = t.File
				state.Run = run
			}

			statesMap[key] = state
		}
	}

	// Add the states to the evaluation context
	t.EvalContext.FileStates = statesMap
	return nil
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
