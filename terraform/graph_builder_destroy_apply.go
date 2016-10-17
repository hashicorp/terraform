package terraform

import (
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// DestroyApplyGraphBuilder implements GraphBuilder and is responsible for
// applying a pure-destroy plan.
//
// This graph builder is very similar to the ApplyGraphBuilder but
// is slightly simpler.
type DestroyApplyGraphBuilder struct {
	// Module is the root module for the graph to build.
	Module *module.Tree

	// Diff is the diff to apply.
	Diff *Diff

	// State is the current state
	State *State

	// Providers is the list of providers supported.
	Providers []string

	// DisableReduce, if true, will not reduce the graph. Great for testing.
	DisableReduce bool
}

// See GraphBuilder
func (b *DestroyApplyGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
	}).Build(path)
}

// See GraphBuilder
func (b *DestroyApplyGraphBuilder) Steps() []GraphTransformer {
	// Custom factory for creating providers.
	providerFactory := func(name string, path []string) GraphNodeProvider {
		return &NodeApplyableProvider{
			NameValue: name,
			PathValue: path,
		}
	}

	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		return &NodeApplyableResource{
			NodeAbstractResource: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the nodes represented in the diff.
		&DiffTransformer{
			Concrete: concreteResource,

			Diff:   b.Diff,
			Module: b.Module,
			State:  b.State,
		},

		// Create orphan output nodes
		&OrphanOutputTransformer{Module: b.Module, State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Module: b.Module},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Destruction ordering. NOTE: For destroys, we don't need to
		// do any CBD stuff, so that is explicitly not here.
		&DestroyEdgeTransformer{Module: b.Module, State: b.State},

		// Create all the providers
		&MissingProviderTransformer{Providers: b.Providers, Factory: providerFactory},
		&ProviderTransformer{},
		&ParentProviderTransformer{},
		&AttachProviderConfigTransformer{Module: b.Module},

		// Add root variables
		&RootVariableTransformer{Module: b.Module},

		// Add module variables
		&ModuleVariableTransformer{Module: b.Module},

		// Add the outputs
		&OutputTransformer{Module: b.Module},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Add the node to fix the state count boundaries
		&CountBoundaryTransformer{},

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
