// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"slices"

	"github.com/hashicorp/hcl/v2"
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

type BindContextOpts interface {
	BindContextOpts(opts *terraform.ContextOpts)
}

// TestFileState is a helper struct that just maps a run block to the state that
// was produced by the execution of that run block.
type TestFileState struct {
	Run   *moduletest.Run
	State *states.State
}

// TestConfigTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
// It also adds a cleanup node for each state that is created by the test runs.
type TestConfigTransformer struct {
	File *moduletest.File
}

func (t *TestConfigTransformer) Transform(g *terraform.Graph) error {
	// This map tracks the state of each run in the file. If multiple runs
	// have the same state key, they will share the same state.
	statesMap := make(map[string]*TestFileState)

	// a root config node that will add the file states to the context
	rootConfigNode := t.addRootConfigNode(g, statesMap)

	cleanupMap := make(map[string]*NodeStateCleanup)
	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}
		key := node.run.GetStateKey()
		if _, exists := statesMap[key]; !exists {
			state := &TestFileState{
				Run:   nil,
				State: states.NewState(),
			}
			statesMap[key] = state
			cleanupMap[key] = &NodeStateCleanup{stateKey: key, file: t.File}
			g.Add(cleanupMap[key])
		}

		// Connect all the test runs to the config node, so that the config node
		// is executed before any of the test runs.
		g.Connect(dag.BasicEdge(node, rootConfigNode))
	}

	// Add a root cleanup node that runs before cleanup nodes for each state.
	// Right now it just simply renders a teardown summary, so as to maintain
	// existing CLI output.
	rootCleanupNode := t.addRootCleanupNode(g)

	for _, v := range g.Vertices() {
		switch node := v.(type) {
		case *NodeTestRun:
			// All the runs that share the same state, must share the same cleanup node,
			// which only executes once after all the runs have completed.
			cleanupNode := cleanupMap[node.run.GetStateKey()]
			g.Connect(dag.BasicEdge(cleanupNode, node))

			// root cleanup node must execute only after all the test runs
			// We could decide to change this behavior in future, and just run the cleanup
			// node after all its dependent runs have completed, but that'll lead
			// to a change in the CLI output.
			g.Connect(dag.BasicEdge(rootCleanupNode, node))
		}
	}

	// connect all cleanup nodes in reverse-sequential order to
	// preserve existing behavior, starting from the root cleanup node,
	// which must run first.
	// TODO: Order by parallelism, so that cleanup nodes that can run in parallel
	added := make(map[string]bool)
	var prev dag.Vertex = rootCleanupNode
	for _, v := range slices.Backward(t.File.Runs) {
		key := v.GetStateKey()
		if _, exists := added[key]; !exists {
			node := cleanupMap[key]
			g.Connect(dag.BasicEdge(node, prev))
			prev = node
			added[key] = true
		}
	}

	return nil
}

func (t *TestConfigTransformer) addRootConfigNode(g *terraform.Graph, statesMap map[string]*TestFileState) *dynamicNode {
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

func (t *TestConfigTransformer) addRootCleanupNode(g *terraform.Graph) *dynamicNode {
	rootCleanupNode := &dynamicNode{
		eval: func(ctx *EvalContext) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			ctx.Renderer().File(t.File, moduletest.TearDown)
			return diags
		},
	}
	g.Add(rootCleanupNode)
	return rootCleanupNode
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

	// First, initialise `previous` and `next`. `previous` contains a backup of
	// the providers from the original config. `next` contains the set of
	// providers that will be used by the test. `next` starts with the set of
	// providers from the original config.
	previous := run.ModuleConfig.Module.ProviderConfigs
	next := make(map[string]*configs.Provider)
	for key, value := range previous {
		next[key] = value
	}

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
