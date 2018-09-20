package terraform

import (
	"log"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// RefreshGraphBuilder implements GraphBuilder and is responsible for building
// a graph for refreshing (updating the Terraform state).
//
// The primary difference between this graph and others:
//
//   * Based on the state since it represents the only resources that
//     need to be refreshed.
//
//   * Ignores lifecycle options since no lifecycle events occur here. This
//     simplifies the graph significantly since complex transforms such as
//     create-before-destroy can be completely ignored.
//
type RefreshGraphBuilder struct {
	// Config is the configuration tree.
	Config *configs.Config

	// State is the prior state
	State *states.State

	// Components is a factory for the plug-in components (providers and
	// provisioners) available for use.
	Components contextComponentFactory

	// Schemas is the repository of schemas we will draw from to analyse
	// the configuration.
	Schemas *Schemas

	// Targets are resources to target
	Targets []addrs.Targetable

	// DisableReduce, if true, will not reduce the graph. Great for testing.
	DisableReduce bool

	// Validate will do structural validation of the graph.
	Validate bool
}

// See GraphBuilder
func (b *RefreshGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: b.Validate,
		Name:     "RefreshGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *RefreshGraphBuilder) Steps() []GraphTransformer {
	// Custom factory for creating providers.
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	concreteManagedResource := func(a *NodeAbstractResource) dag.Vertex {
		return &NodeRefreshableManagedResource{
			NodeAbstractResource: a,
		}
	}

	concreteManagedResourceInstance := func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodeRefreshableManagedResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	concreteResourceInstanceDeposed := func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		// The "Plan" node type also handles refreshing behavior.
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey: key,
		}
	}

	concreteDataResource := func(a *NodeAbstractResource) dag.Vertex {
		return &NodeRefreshableDataResource{
			NodeAbstractResource: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the managed resources that aren't in the state, but only if
		// we have a state already. No resources in state means there's not
		// anything to refresh.
		func() GraphTransformer {
			if b.State.HasResources() {
				return &ConfigTransformer{
					Concrete:   concreteManagedResource,
					Config:     b.Config,
					Unique:     true,
					ModeFilter: true,
					Mode:       addrs.ManagedResourceMode,
				}
			}
			log.Println("[TRACE] No managed resources in state during refresh; skipping managed resource transformer")
			return nil
		}(),

		// Creates all the data resources that aren't in the state. This will also
		// add any orphans from scaling in as destroy nodes.
		&ConfigTransformer{
			Concrete:   concreteDataResource,
			Config:     b.Config,
			Unique:     true,
			ModeFilter: true,
			Mode:       addrs.DataResourceMode,
		},

		// Add any fully-orphaned resources from config (ones that have been
		// removed completely, not ones that are just orphaned due to a scaled-in
		// count.
		&OrphanResourceTransformer{
			Concrete: concreteManagedResourceInstance,
			State:    b.State,
			Config:   b.Config,
		},

		// We also need nodes for any deposed instance objects present in the
		// state, so we can check if they still exist. (This intentionally
		// skips creating nodes for _current_ objects, since ConfigTransformer
		// created nodes that will do that during DynamicExpand.)
		&StateTransformer{
			ConcreteDeposed: concreteResourceInstanceDeposed,
			State:           b.State,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// Add root variables
		&RootVariableTransformer{Config: b.Config},

		// Add the local values
		&LocalTransformer{Config: b.Config},

		// Add the outputs
		&OutputTransformer{Config: b.Config},

		// Add module variables
		&ModuleVariableTransformer{Config: b.Config},

		TransformProviders(b.Components.ResourceProviders(), concreteProvider, b.Config),

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Schemas: b.Schemas},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Target
		&TargetsTransformer{
			Targets: b.Targets,

			// Resource nodes from config have not yet been expanded for
			// "count", so we must apply targeting without indices. Exact
			// targeting will be dealt with later when these resources
			// DynamicExpand.
			IgnoreIndices: true,
		},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Single root
		&RootTransformer{},
	}

	if !b.DisableReduce {
		// Perform the transitive reduction to make our graph a bit
		// more sane if possible (it usually is possible).
		steps = append(steps, &TransitiveReductionTransformer{})
	}

	return steps
}
