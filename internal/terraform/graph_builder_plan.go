package terraform

import (
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

	// RootVariableValues are the raw input values for root input variables
	// given by the caller, which we'll resolve into final values as part
	// of the plan walk.
	RootVariableValues InputValues

	// Plugins is a library of plug-in components (providers and
	// provisioners) available for use.
	Plugins *contextPlugins

	// Targets are resources to target
	Targets []addrs.Targetable

	// ForceReplace are resource instances where if we would normally have
	// generated a NoOp or Update action then we'll force generating a replace
	// action instead. Create and Delete actions are not affected.
	ForceReplace []addrs.AbsResourceInstance

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
	CustomConcrete                  bool
	ConcreteProvider                ConcreteProviderNodeFunc
	ConcreteResource                ConcreteResourceNodeFunc
	ConcreteResourceInstance        ConcreteResourceInstanceNodeFunc
	ConcreteResourceOrphan          ConcreteResourceInstanceNodeFunc
	ConcreteResourceInstanceDeposed ConcreteResourceInstanceDeposedNodeFunc
	ConcreteModule                  ConcreteModuleNodeFunc

	// destroy is set to true when create a full destroy plan.
	destroy bool
}

// See GraphBuilder
func (b *PlanGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "PlanGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *PlanGraphBuilder) Steps() []GraphTransformer {
	b.init()

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Config:   b.Config,
			skip:     b.destroy,
		},

		// Add dynamic values
		&RootVariableTransformer{Config: b.Config, RawValues: b.RootVariableValues},
		&ModuleVariableTransformer{Config: b.Config},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{
			Config:            b.Config,
			RefreshOnly:       b.skipPlanChanges,
			removeRootOutputs: b.destroy,
		},

		// Add orphan resources
		&OrphanResourceInstanceTransformer{
			Concrete: b.ConcreteResourceOrphan,
			State:    b.State,
			Config:   b.Config,
			skip:     b.destroy,
		},

		// We also need nodes for any deposed instance objects present in the
		// state, so we can plan to destroy them. (This intentionally
		// skips creating nodes for _current_ objects, since ConfigTransformer
		// created nodes that will do that during DynamicExpand.)
		&StateTransformer{
			ConcreteCurrent: b.ConcreteResourceInstance,
			ConcreteDeposed: b.ConcreteResourceInstanceDeposed,
			State:           b.State,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Create orphan output nodes
		&OrphanOutputTransformer{Config: b.Config, State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// add providers
		transformProviders(b.ConcreteProvider, b.Config),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Plugins: b.Plugins, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Concrete: b.ConcreteModule, Config: b.Config},

		&ReferenceTransformer{},

		&AttachDependenciesTransformer{},

		// Make sure data sources are aware of any depends_on from the
		// configuration
		&attachDataResourceDependsOnTransformer{},

		// DestroyEdgeTransformer is only required during a plan so that the
		// TargetsTransformer can determine which nodes to keep in the graph.
		&DestroyEdgeTransformer{},

		// Target
		&TargetsTransformer{Targets: b.Targets},

		// Detect when create_before_destroy must be forced on for a particular
		// node due to dependency edges, to avoid graph cycles during apply.
		&ForcedCBDTransformer{},

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

	b.ConcreteResourceInstanceDeposed = func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey:                   key,

			skipRefresh:     b.skipRefresh,
			skipPlanChanges: b.skipPlanChanges,
		}
	}

}
