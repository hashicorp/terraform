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

	// Providers is the list of providers supported.
	Providers []string
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

		TransformProviders(b.Providers, concreteProvider, config),

		// This validates that the providers only depend on variables
		&ImportProviderValidateTransformer{},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Single root
		&RootTransformer{},

		// Optimize
		&TransitiveReductionTransformer{},
	}

	return steps
}
