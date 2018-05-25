package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// PlanGraphBuilder implements GraphBuilder and is responsible for building
// a graph for planning (creating a Terraform Diff).
//
// The primary difference between this graph and others:
//
//   * Based on the config since it represents the target state
//
//   * Ignores lifecycle options since no lifecycle events occur here. This
//     simplifies the graph significantly since complex transforms such as
//     create-before-destroy can be completely ignored.
//
type PlanGraphBuilder struct {
	// Config is the configuration tree to build a plan from.
	Config *configs.Config

	// State is the current state
	State *State

	// Components is a factory for the plug-in components (providers and
	// provisioners) available for use.
	Components contextComponentFactory

	// Targets are resources to target
	Targets []addrs.Targetable

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
	ConcreteResourceOrphan ConcreteResourceInstanceNodeFunc

	once sync.Once
}

// See GraphBuilder
func (b *PlanGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: b.Validate,
		Name:     "PlanGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *PlanGraphBuilder) Steps() []GraphTransformer {
	b.once.Do(b.init)

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Config:   b.Config,
		},

		// Add the local values
		&LocalTransformer{Config: b.Config},

		// Add the outputs
		&OutputTransformer{Config: b.Config},

		// Add orphan resources
		&OrphanResourceTransformer{
			Concrete: b.ConcreteResourceOrphan,
			State:    b.State,
			Config:   b.Config,
		},

		// Create orphan output nodes
		&OrphanOutputTransformer{
			Config: b.Config,
			State:  b.State,
		},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Add root variables
		&RootVariableTransformer{Config: b.Config},

		&MissingProvisionerTransformer{Provisioners: b.Components.ResourceProvisioners()},

		&AttachSchemaTransformer{Components: b.Components},

		// Add module variables
		&ModuleVariableTransformer{
			Config: b.Config,
		},

		TransformProviders(b.Components.ResourceProviders(), b.ConcreteProvider, b.Config),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must be before ReferenceTransformer, since schema is required to
		// extract references from config.
		&AttachSchemaTransformer{Components: b.Components},

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

func (b *PlanGraphBuilder) init() {
	// Do nothing if the user requests customizing the fields
	if b.CustomConcrete {
		return
	}

	b.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	b.ConcreteResource = func(a *NodeAbstractResource) dag.Vertex {
		return &NodePlannableResource{
			NodeAbstractResource: a,
			Components:           b.Components,
		}
	}

	b.ConcreteResourceOrphan = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
		}
	}
}
