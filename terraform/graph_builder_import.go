package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

// ImportGraphBuilder implements GraphBuilder and is responsible for building
// a graph for importing resources into Terraform. This is a much, much
// simpler graph than a normal configuration graph.
type ImportGraphBuilder struct {
	// ImportTargets are the list of resources to import.
	ImportTargets []*ImportTarget

	// Module is a configuration to build the graph from. See ImportOpts.Config.
	Config *configs.Config

	// Components is the factory for our available plugin components.
	Components contextComponentFactory

	// Schemas is the repository of schemas we will draw from to analyse
	// the configuration.
	Schemas *Schemas
}

// Build builds the graph according to the steps returned by Steps.
func (b *ImportGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
		Name:     "ImportGraphBuilder",
	}).Build(path)
}

// Steps returns the ordered list of GraphTransformers that must be executed
// to build a complete graph.
func (b *ImportGraphBuilder) Steps() []GraphTransformer {
	// Get the module. If we don't have one, we just use an empty tree
	// so that the transform still works but does nothing.
	config := b.Config
	if config == nil {
		config = configs.NewEmptyConfig()
	}

	// Custom factory for creating providers.
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Create all our resources from the configuration and state
		&ConfigTransformer{Config: config},

		// Add the import steps
		&ImportStateTransformer{Targets: b.ImportTargets},

		// Add root variables
		&RootVariableTransformer{Config: b.Config},

		TransformProviders(b.Components.ResourceProviders(), concreteProvider, config),

		// This validates that the providers only depend on variables
		&ImportProviderValidateTransformer{},

		// Add the local values
		&LocalTransformer{Config: b.Config},

		// Add the outputs
		&OutputTransformer{Config: b.Config},

		// Add module variables
		&ModuleVariableTransformer{Config: b.Config},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Schemas: b.Schemas},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Single root
		&RootTransformer{},

		// Optimize
		&TransitiveReductionTransformer{},
	}

	return steps
}
