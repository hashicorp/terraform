package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	State *states.State

	// Components is a factory for the plug-in components (providers and
	// provisioners) available for use.
	Components contextComponentFactory

	// Schemas is the repository of schemas we will draw from to analyse
	// the configuration.
	Schemas *Schemas

	// Targets are resources to target
	Targets []addrs.Targetable

	// ForceReplace are resource instances where if we would normally have
	// generated a NoOp or Update action then we'll force generating a replace
	// action instead. Create and Delete actions are not affected.
	ForceReplace []addrs.AbsResourceInstance

	// Validate will do structural validation of the graph.
	Validate bool

	// skipRefresh indicates that we should skip refreshing managed resources
	skipRefresh bool

	// skipPlanChanges indicates that we should skip the step of comparing
	// prior state with configuration and generating planned changes to
	// resource instances. (This is for the "refresh only" planning mode,
	// where we _only_ do the refresh step.)
	skipPlanChanges bool

	// CustomConcrete can be set to customize the node types created
	// for various parts of the plan. This is useful in order to customize
	// the plan behavior.
	CustomConcrete         bool
	ConcreteProvider       ConcreteProviderNodeFunc
	ConcreteResource       ConcreteResourceNodeFunc
	ConcreteResourceOrphan ConcreteResourceInstanceNodeFunc
	ConcreteModule         ConcreteModuleNodeFunc

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

	concreteResourceInstanceDeposed := func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey:                   key,

			skipRefresh:     b.skipRefresh,
			skipPlanChanges: b.skipPlanChanges,
		}
	}

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Config:   b.Config,
		},

		// Add dynamic values
		&RootVariableTransformer{Config: b.Config},
		&ModuleVariableTransformer{Config: b.Config},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{Config: b.Config},

		// Add orphan resources
		&OrphanResourceInstanceTransformer{
			Concrete: b.ConcreteResourceOrphan,
			State:    b.State,
			Config:   b.Config,
		},

		// We also need nodes for any deposed instance objects present in the
		// state, so we can plan to destroy them. (This intentionally
		// skips creating nodes for _current_ objects, since ConfigTransformer
		// created nodes that will do that during DynamicExpand.)
		&StateTransformer{
			ConcreteDeposed: concreteResourceInstanceDeposed,
			State:           b.State,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Create orphan output nodes
		&OrphanOutputTransformer{Config: b.Config, State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// add providers
		TransformProviders(b.Components.ResourceProviders(), b.ConcreteProvider, b.Config),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Schemas: b.Schemas, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Concrete: b.ConcreteModule, Config: b.Config},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},
		&AttachDependenciesTransformer{},

		// Make sure data sources are aware of any depends_on from the
		// configuration
		&attachDataResourceDependsOnTransformer{},

		// Target
		&TargetsTransformer{Targets: b.Targets},

		// Detect when create_before_destroy must be forced on for a particular
		// node due to dependency edges, to avoid graph cycles during apply.
		&ForcedCBDTransformer{},

		// Add the node to fix the state count boundaries
		&CountBoundaryTransformer{
			Config: b.Config,
		},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Close the root module
		&CloseRootModuleTransformer{},

		// Perform the transitive reduction to make our graph a bit
		// more understandable if possible (it usually is possible).
		&TransitiveReductionTransformer{},
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
		return &nodeExpandPlannableResource{
			NodeAbstractResource: a,
			skipRefresh:          b.skipRefresh,
			skipPlanChanges:      b.skipPlanChanges,
			forceReplace:         b.ForceReplace,
		}
	}

	b.ConcreteResourceOrphan = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  b.skipRefresh,
			skipPlanChanges:              b.skipPlanChanges,
		}
	}
}
