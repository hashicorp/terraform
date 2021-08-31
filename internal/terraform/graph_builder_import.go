package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ImportGraphBuilder implements GraphBuilder and is responsible for building
// a graph for importing resources into Terraform. This is a much, much
// simpler graph than a normal configuration graph.
type ImportGraphBuilder struct {
	// ImportTargets are the list of resources to import.
	ImportTargets []*ImportTarget

	// Module is a configuration to build the graph from. See ImportOpts.Config.
	Config *configs.Config

	// Plugins is a library of plug-in components (providers and
	// provisioners) available for use.
	Plugins *contextPlugins
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

		// Add dynamic values
		&RootVariableTransformer{Config: b.Config},
		&ModuleVariableTransformer{Config: b.Config},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{Config: b.Config},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// Add the import steps
		&ImportStateTransformer{Targets: b.ImportTargets, Config: b.Config},

		transformProviders(concreteProvider, config),

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Plugins: b.Plugins, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Config: b.Config},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Make sure data sources are aware of any depends_on from the
		// configuration
		&attachDataResourceDependsOnTransformer{},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Close root module
		&CloseRootModuleTransformer{},

		// Optimize
		&TransitiveReductionTransformer{},
	}

	return steps
}
