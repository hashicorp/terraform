package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// DestroyPlanGraphBuilder implements GraphBuilder and is responsible for
// planning a pure-destroy.
//
// Planning a pure destroy operation is simple because we can ignore most
// ordering configuration and simply reverse the state. This graph mainly
// exists for targeting, because we need to walk the destroy dependencies to
// ensure we plan the required resources. Without the requirement for
// targeting, the plan could theoretically be created directly from the state.
type DestroyPlanGraphBuilder struct {
	// Config is the configuration tree to build the plan from.
	Config *configs.Config

	// State is the current state
	State *states.State

	// Plugins is a library of plug-in components (providers and
	// provisioners) available for use.
	Plugins *contextPlugins

	// Targets are resources to target
	Targets []addrs.Targetable

	// Validate will do structural validation of the graph.
	Validate bool

	// If set, skipRefresh will cause us stop skip refreshing any existing
	// resource instances as part of our planning. This will cause us to fail
	// to detect if an object has already been deleted outside of Terraform.
	skipRefresh bool
}

// See GraphBuilder
func (b *DestroyPlanGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: b.Validate,
		Name:     "DestroyPlanGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *DestroyPlanGraphBuilder) Steps() []GraphTransformer {
	concreteResourceInstance := func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlanDestroyableResourceInstance{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  b.skipRefresh,
		}
	}
	concreteResourceInstanceDeposed := func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey:                   key,
			skipRefresh:                  b.skipRefresh,
		}
	}

	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Creates nodes for the resource instances tracked in the state.
		&StateTransformer{
			ConcreteCurrent: concreteResourceInstance,
			ConcreteDeposed: concreteResourceInstanceDeposed,
			State:           b.State,
		},

		// Create the delete changes for root module outputs.
		&OutputTransformer{
			Config:  b.Config,
			Destroy: true,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		transformProviders(concreteProvider, b.Config),

		// Destruction ordering. We require this only so that
		// targeting below will prune the correct things.
		&DestroyEdgeTransformer{
			Config: b.Config,
			State:  b.State,
		},

		&TargetsTransformer{Targets: b.Targets},

		// Close opened plugin connections
		&CloseProviderTransformer{},

		// Close the root module
		&CloseRootModuleTransformer{},

		&TransitiveReductionTransformer{},
	}

	return steps
}
