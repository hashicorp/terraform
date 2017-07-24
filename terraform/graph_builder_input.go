package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// InputGraphBuilder implements GraphBuilder and is responsible for building
// a graph for input.
type InputGraphBuilder struct {
	// Module is the root module for the graph to build.
	Module *module.Tree

	// State is the current state
	State *State

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

	// CustomConcrete can be set to customize the node types created
	// for various parts of the plan. This is useful in order to customize
	// the plan behavior.
	CustomConcrete         bool
	ConcreteProvider       ConcreteProviderNodeFunc
	ConcreteResource       ConcreteResourceNodeFunc
	ConcreteResourceOrphan ConcreteResourceNodeFunc

	once sync.Once
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
	b.once.Do(b.init)

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Module:   b.Module,
		},

		// Add the outputs
		&OutputTransformer{Module: b.Module},

		// Add orphan resources
		&OrphanResourceTransformer{
			Concrete: b.ConcreteResourceOrphan,
			State:    b.State,
			Module:   b.Module,
		},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Module: b.Module},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Add root variables
		&RootVariableTransformer{Module: b.Module},

		// Create all the providers
		&MissingProviderTransformer{Providers: b.Providers, Concrete: b.ConcreteProvider},
		&ProviderTransformer{},
		&DisableProviderTransformer{},
		&ParentProviderTransformer{},
		&AttachProviderConfigTransformer{Module: b.Module},

		// Provisioner-related transformations. Only add these if requested.
		GraphTransformIf(
			func() bool { return b.Provisioners != nil },
			GraphTransformMulti(
				&MissingProvisionerTransformer{Provisioners: b.Provisioners},
				&ProvisionerTransformer{},
			),
		),

		// Add module variables
		&ModuleVariableTransformer{Module: b.Module},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Add the node to fix the state count boundaries
		&CountBoundaryTransformer{},

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

func (b *InputGraphBuilder) init() {
	// We're going to customize the concrete functions
	b.CustomConcrete = true

	// Set the provider to the normal provider. This will ask for input.
	b.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}
}
