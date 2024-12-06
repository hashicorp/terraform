// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ApplyGraphBuilder implements GraphBuilder and is responsible for building
// a graph for applying a Terraform diff.
//
// Because the graph is built from the diff (vs. the config or state),
// this helps ensure that the apply-time graph doesn't modify any resources
// that aren't explicitly in the diff. There are other scenarios where the
// diff can be deviated, so this is just one layer of protection.
type ApplyGraphBuilder struct {
	// Config is the configuration tree that the diff was built from.
	Config *configs.Config

	// Changes describes the changes that we need apply.
	Changes *plans.ChangesSrc

	// DeferredChanges describes the changes that were deferred during the plan
	// and should not be applied.
	DeferredChanges []*plans.DeferredResourceInstanceChangeSrc

	// State is the current state
	State *states.State

	// RootVariableValues are the root module input variables captured as
	// part of the plan object, which we must reproduce in the apply step
	// to get a consistent result.
	RootVariableValues InputValues

	// ExternalProviderConfigs are pre-initialized root module provider
	// configurations that the graph builder should assume will be available
	// immediately during the subsequent plan walk, without any explicit
	// initialization step.
	ExternalProviderConfigs map[addrs.RootProviderConfig]providers.Interface

	// Plugins is a library of the plug-in components (providers and
	// provisioners) available for use.
	Plugins *contextPlugins

	// Targets are resources to target. This is only required to make sure
	// unnecessary outputs aren't included in the apply graph. The plan
	// builder successfully handles targeting resources. In the future,
	// outputs should go into the diff so that this is unnecessary.
	Targets []addrs.Targetable

	// ForceReplace are the resource instance addresses that the user
	// requested to force replacement for when creating the plan, if any.
	// The apply step refers to these as part of verifying that the planned
	// actions remain consistent between plan and apply.
	ForceReplace []addrs.AbsResourceInstance

	// Plan Operation this graph will be used for.
	Operation walkOperation

	// ExternalReferences allows the external caller to pass in references to
	// nodes that should not be pruned even if they are not referenced within
	// the actual graph.
	ExternalReferences []*addrs.Reference

	// Overrides provides the set of overrides supplied by the testing
	// framework.
	Overrides *mocking.Overrides
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "ApplyGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Steps() []GraphTransformer {
	// Custom factory for creating providers.
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		return &nodeExpandApplyableResource{
			NodeAbstractResource: a,
		}
	}

	concreteResourceInstance := func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodeApplyableResourceInstance{
			NodeAbstractResourceInstance: a,
			forceReplace:                 b.ForceReplace,
		}
	}

	steps := []GraphTransformer{
		// Creates all the resources represented in the config. During apply,
		// we use this just to ensure that the whole-resource metadata is
		// updated to reflect things such as whether the count argument is
		// set in config, or which provider configuration manages each resource.
		&ConfigTransformer{
			Concrete: concreteResource,
			Config:   b.Config,
		},

		// Add dynamic values
		&RootVariableTransformer{
			Config:       b.Config,
			RawValues:    b.RootVariableValues,
			DestroyApply: b.Operation == walkDestroy,
		},
		&ModuleVariableTransformer{
			Config:       b.Config,
			DestroyApply: b.Operation == walkDestroy,
		},
		&variableValidationTransformer{},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{
			Config:     b.Config,
			Destroying: b.Operation == walkDestroy,
			Overrides:  b.Overrides,
		},

		// Creates all the resource instances represented in the diff, along
		// with dependency edges against the whole-resource nodes added by
		// ConfigTransformer above.
		&DiffTransformer{
			Concrete: concreteResourceInstance,
			State:    b.State,
			Changes:  b.Changes,
			Config:   b.Config,
		},

		// Creates nodes for all the deferred changes.
		&DeferredTransformer{
			DeferredChanges: b.DeferredChanges,
		},

		// Add nodes and edges for check block assertions. Check block data
		// sources were added earlier.
		&checkTransformer{
			Config:    b.Config,
			Operation: b.Operation,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Create orphan output nodes
		&OrphanOutputTransformer{Config: b.Config, State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// add providers
		transformProviders(concreteProvider, b.Config, b.ExternalProviderConfigs),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Plugins: b.Plugins, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Config: b.Config},

		// Plug in any external references.
		&ExternalReferenceTransformer{
			ExternalReferences: b.ExternalReferences,
		},

		// Connect references so ordering is correct
		&ReferenceTransformer{},
		&AttachDependenciesTransformer{},

		&OutputReferencesTransformer{},

		// Nested data blocks should be loaded after every other resource has
		// done its thing.
		&checkStartTransformer{Config: b.Config, Operation: b.Operation},

		// Detect when create_before_destroy must be forced on for a particular
		// node due to dependency edges, to avoid graph cycles during apply.
		//
		// FIXME: this should not need to be recalculated during apply.
		// Currently however, the instance object which stores the planned
		// information is lost for newly created instances because it contains
		// no state value, and we end up recalculating CBD for all nodes.
		&ForcedCBDTransformer{},

		// Destruction ordering
		&DestroyEdgeTransformer{
			Changes:   b.Changes,
			Operation: b.Operation,
		},
		&CBDEdgeTransformer{
			Config: b.Config,
			State:  b.State,
		},

		// In a destroy, we need to remove configuration nodes that are not used
		// at all, as they may not be able to evaluate. These include variables,
		// locals, and instance expanders.
		&pruneUnusedNodesTransformer{
			skip: b.Operation != walkDestroy,
		},

		// Target
		&TargetsTransformer{Targets: b.Targets},

		// Close any ephemeral resource instances.
		&ephemeralResourceCloseTransformer{},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// close the root module
		&CloseRootModuleTransformer{},

		// Perform the transitive reduction to make our graph a bit
		// more understandable if possible (it usually is possible).
		&TransitiveReductionTransformer{},
	}

	return steps
}
