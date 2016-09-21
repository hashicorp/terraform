package terraform

import (
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// ApplyGraphBuilder implements GraphBuilder and is responsible for building
// a graph for applying a Terraform diff.
//
// Because the graph is built from the diff (vs. the config or state),
// this helps ensure that the apply-time graph doesn't modify any resources
// that aren't explicitly in the diff. There are other scenarios where the
// diff can be deviated, so this is just one layer of protection.
type ApplyGraphBuilder struct {
	// Module is the root module for the graph to build.
	Module *module.Tree

	// Diff is the diff to apply.
	Diff *Diff

	// State is the current state
	State *State

	// Providers is the list of providers supported.
	Providers []string

	// Provisioners is the list of provisioners supported.
	Provisioners []string
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
	}).Build(path)
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Steps() []GraphTransformer {
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

		// Destruction ordering
		&DestroyEdgeTransformer{Module: b.Module, State: b.State},

		// Create all the providers
		&MissingProviderTransformer{Providers: b.Providers, Factory: providerFactory},
		&ProviderTransformer{},
		&ParentProviderTransformer{},
		&AttachProviderConfigTransformer{Module: b.Module},

		// Provisioner-related transformations
		&MissingProvisionerTransformer{Provisioners: b.Provisioners},
		&ProvisionerTransformer{},

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

		// Perform the transitive reduction to make our graph a bit
		// more sane if possible (it usually is possible).
		&TransitiveReductionTransformer{},
	}

	return steps
}
