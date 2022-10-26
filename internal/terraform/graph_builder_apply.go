package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
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
	Changes *plans.Changes

	// State is the current state
	State *states.State

	// RootVariableValues are the root module input variables captured as
	// part of the plan object, which we must reproduce in the apply step
	// to get a consistent result.
	RootVariableValues InputValues

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
		&RootVariableTransformer{Config: b.Config, RawValues: b.RootVariableValues},
		&ModuleVariableTransformer{Config: b.Config},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{
			Config:       b.Config,
			ApplyDestroy: b.Operation == walkDestroy,
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

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Create orphan output nodes
		&OrphanOutputTransformer{Config: b.Config, State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// add providers
		transformProviders(concreteProvider, b.Config),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Plugins: b.Plugins, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Config: b.Config},

		// Connect references so ordering is correct
		&ReferenceTransformer{},
		&AttachDependenciesTransformer{},

		// Detect when create_before_destroy must be forced on for a particular
		// node due to dependency edges, to avoid graph cycles during apply.
		&ForcedCBDTransformer{},

		// Destruction ordering
		&DestroyEdgeTransformer{
			Changes: b.Changes,
		},
		&CBDEdgeTransformer{
			Config: b.Config,
			State:  b.State,
		},

		// We need to remove configuration nodes that are not used at all, as
		// they may not be able to evaluate, especially during destroy.
		// These include variables, locals, and instance expanders.
		&pruneUnusedNodesTransformer{},

		// Target
		&TargetsTransformer{Targets: b.Targets},

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
