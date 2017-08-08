package terraform

import (
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// InputGraphBuilder implements GraphBuilder and is responsible for building
// a graph for input.
//
// It is a simplified version of the PlanGraphBuilder, adding only what is
// required to locate missing variables and provider values.
type InputGraphBuilder struct {
	// Module is the root module for the graph to build.
	Module *module.Tree

	// Providers is the list of providers supported.
	Providers []string

	// Provisioners is the list of provisioners supported.
	Provisioners []string

	// Targets are resources to target
	Targets []string

	// DisableReduce, if true, will not reduce the graph. Great for testing.
	DisableReduce bool

	// Validate will do structural validation of the graph.
	Validate bool
}

// See GraphBuilder
func (b *InputGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: b.Validate,
		Name:     "InputGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *InputGraphBuilder) Steps() []GraphTransformer {
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{Module: b.Module},

		// Add the outputs
		&OutputTransformer{Module: b.Module},

		// Add root variables
		&RootVariableTransformer{Module: b.Module},

		// Create all the providers
		&MissingProviderTransformer{Providers: b.Providers, Concrete: concreteProvider},
		&ProviderTransformer{},
		&DisableProviderTransformer{},
		&ParentProviderTransformer{},
		&AttachProviderConfigTransformer{Module: b.Module},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Add the node to fix the state count boundaries
		&CountBoundaryTransformer{},

		// Target
		&TargetsTransformer{
			Targets:       b.Targets,
			IgnoreIndices: true,
		},

		// Close opened plugin connections
		&CloseProviderTransformer{},
		&CloseProvisionerTransformer{},

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
