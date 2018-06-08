package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
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
	// Module is the root module for the graph to build.
	Module *module.Tree

	// State is the current state
	State *State

	// Providers is the list of providers supported.
	Providers []string

	// Targets are resources to target
	Targets []string

	// DisableReduce, if true, will not reduce the graph. Great for testing.
	DisableReduce bool

	// Validate will do structural validation of the graph.
	Validate bool
}

// See GraphBuilder
func (b *RefreshGraphBuilder) Build(path []string) (*Graph, error) {
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
			NodeAbstractCountResource: &NodeAbstractCountResource{
				NodeAbstractResource: a,
			},
		}
	}

	concreteManagedResourceInstance := func(a *NodeAbstractResource) dag.Vertex {
		return &NodeRefreshableManagedResourceInstance{
			NodeAbstractResource: a,
		}
	}

	concreteDataResource := func(a *NodeAbstractResource) dag.Vertex {
		return &NodeRefreshableDataResource{
			NodeAbstractCountResource: &NodeAbstractCountResource{
				NodeAbstractResource: a,
			},
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
					Module:     b.Module,
					Unique:     true,
					ModeFilter: true,
					Mode:       config.ManagedResourceMode,
				}
			}
			log.Println("[TRACE] No managed resources in state during refresh, skipping managed resource transformer")
			return nil
		}(),

		// Creates all the data resources that aren't in the state. This will also
		// add any orphans from scaling in as destroy nodes.
		&ConfigTransformer{
			Concrete:   concreteDataResource,
			Module:     b.Module,
			Unique:     true,
			ModeFilter: true,
			Mode:       config.DataResourceMode,
		},

		// Add any fully-orphaned resources from config (ones that have been
		// removed completely, not ones that are just orphaned due to a scaled-in
		// count.
		&OrphanResourceTransformer{
			Concrete: concreteManagedResourceInstance,
			State:    b.State,
			Module:   b.Module,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Module: b.Module},

		// Add root variables
		&RootVariableTransformer{Module: b.Module},

		TransformProviders(b.Providers, concreteProvider, b.Module),

		// Add the local values
		&LocalTransformer{Module: b.Module},

		// Add the outputs
		&OutputTransformer{Module: b.Module},

		// Add module variables
		&ModuleVariableTransformer{Module: b.Module},

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
