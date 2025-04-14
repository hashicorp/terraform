// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type GraphNodeExecutable interface {
	Execute(ctx *EvalContext) tfdiags.Diagnostics
}

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

				log.Printf("[TRACE] TestConfigTransformer.Transform: set initial state for state key %q using backend of type %T declared at %s", key, be, bc.Backend.DeclRange)
				state = &TestFileState{
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

	// Add a helper node to the graph. This node is responsible for
	// injecting the initialized file states into the evaluation context
	// before any test runs are executed.
	configSetterNode := t.addRootConfigNode(g, statesMap)

	// Iterate through all the test run nodes in the graph and connect them to
	// the root configuration node. This ensures that the root configuration node
	// is executed first, setting up the context for the test runs.
	for node := range dag.SelectSeq(g.VerticesSeq(), runFilter) {
		g.Connect(dag.BasicEdge(node, configSetterNode))
	}

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

func (t *TestStateTransformer) addRootConfigNode(g *terraform.Graph, statesMap map[string]*TestFileState) *dynamicNode {
	rootConfigNode := &dynamicNode{
		eval: func(ctx *EvalContext) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			ctx.FileStates = statesMap
			return diags
		},
	}
	g.Add(rootConfigNode)
	return rootConfigNode
}

// TransformConfigForRun transforms the run's module configuration to include
// the providers and variables from its block and the test file.
//
// In practice, this actually just means performing some surgery on the
// available providers. We want to copy the relevant providers from the test
// file into the configuration. We also want to process the providers so they
// use variables from the file instead of variables from within the test file.
func TransformConfigForRun(ctx *EvalContext, run *moduletest.Run, file *moduletest.File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Currently, we only need to override the provider settings.
	//
	// We can have a set of providers defined within the config, we can also
	// have a set of providers defined within the test file. Then the run can
	// also specify a set of overrides that tell Terraform exactly which
	// providers from the test file to apply into the config.
	//
	// The process here is as follows:
	//   1. Take all the providers in the original config keyed by name.alias,
	//      we call this `previous`
	//   2. Copy them all into a new map, we call this `next`.
	//   3a. If the run has configuration specifying provider overrides, we copy
	//       only the specified providers from the test file into `next`. While
	//       doing this we ensure to preserve the name and alias from the
	//       original config.
	//   3b. If the run has no override configuration, we copy all the providers
	//       from the test file into `next`, overriding all providers with name
	//       collisions from the original config.
	//   4. We then modify the original configuration so that the providers it
	//      holds are the combination specified by the original config, the test
	//      file and the run file.
	//   5. We then return a function that resets the original config back to
	//      its original state. This can be called by the surrounding test once
	//      completed so future run blocks can safely execute.

	// First, initialise the providers which we are going to use for the test.
	// It starts with the providers from the original module config, and then we'll
	// overwrite them with the providers from the test file.
	next := make(map[string]*configs.Provider)
	maps.Copy(next, run.ModuleConfig.Module.ProviderConfigs)

	runOutputs := ctx.GetOutputs()

	if len(run.Config.Providers) > 0 {
		// Then we'll only copy over and overwrite the specific providers asked
		// for by this run block.
		for _, ref := range run.Config.Providers {
			testProvider, ok := file.Config.Providers[ref.InParent.String()]
			if !ok {
				// Then this reference was invalid as we didn't have the
				// specified provider in the parent. This should have been
				// caught earlier in validation anyway so is unlikely to happen.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Missing provider definition for %s", ref.InParent.String()),
					Detail:   "This provider block references a provider definition that does not exist.",
					Subject:  ref.InParent.NameRange.Ptr(),
				})
				continue
			}

			next[ref.InChild.String()] = &configs.Provider{
				Name:       ref.InChild.Name,
				NameRange:  ref.InChild.NameRange,
				Alias:      ref.InChild.Alias,
				AliasRange: ref.InChild.AliasRange,
				Config: &hcltest.ProviderConfig{
					Original:            testProvider.Config,
					VariableCache:       ctx.GetCache(run),
					AvailableRunOutputs: runOutputs,
				},
				Mock:      testProvider.Mock,
				MockData:  testProvider.MockData,
				DeclRange: testProvider.DeclRange,
			}
		}
	} else {
		// Otherwise, let's copy over and overwrite all providers specified by
		// the test file itself.
		for key, provider := range file.Config.Providers {

			if !ctx.ProviderExists(run, key) {
				// Then we don't actually need this provider for this
				// configuration, so skip it.
				continue
			}

			next[key] = &configs.Provider{
				Name:       provider.Name,
				NameRange:  provider.NameRange,
				Alias:      provider.Alias,
				AliasRange: provider.AliasRange,
				Config: &hcltest.ProviderConfig{
					Original:            provider.Config,
					VariableCache:       ctx.GetCache(run),
					AvailableRunOutputs: runOutputs,
				},
				Mock:      provider.Mock,
				MockData:  provider.MockData,
				DeclRange: provider.DeclRange,
			}
		}
	}

	run.ModuleConfig.Module.ProviderConfigs = next
	return diags
}
