// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// PlanGraphBuilder is a GraphBuilder implementation that builds a graph for
// planning and for other "plan-like" operations which don't require an
// already-calculated plan as input.
//
// Unlike the apply graph builder, this graph builder:
//
//   - Makes its decisions primarily based on the given configuration, which
//     represents the desired state.
//
//   - Ignores certain lifecycle concerns like create_before_destroy, because
//     those are only important once we already know what action we're planning
//     to take against a particular resource instance.
type PlanGraphBuilder struct {
	// Config is the configuration tree to build a plan from.
	Config *configs.Config

	// State is the current state
	State *states.State

	// RootVariableValues are the raw input values for root input variables
	// given by the caller, which we'll resolve into final values as part
	// of the plan walk.
	RootVariableValues InputValues

	// ExternalProviderConfigs are pre-initialized root module provider
	// configurations that the graph builder should assume will be available
	// immediately during the subsequent plan walk, without any explicit
	// initialization step.
	ExternalProviderConfigs map[addrs.RootProviderConfig]providers.Interface

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

	// preDestroyRefresh indicates that we are executing the refresh which
	// happens immediately before a destroy plan, which happens to use the
	// normal planing mode so skipPlanChanges cannot be set.
	preDestroyRefresh bool

	// skipPlanChanges indicates that we should skip the step of comparing
	// prior state with configuration and generating planned changes to
	// resource instances. (This is for the "refresh only" planning mode,
	// where we _only_ do the refresh step.)
	skipPlanChanges bool

	ConcreteProvider                ConcreteProviderNodeFunc
	ConcreteResource                ConcreteResourceNodeFunc
	ConcreteResourceInstance        ConcreteResourceInstanceNodeFunc
	ConcreteResourceOrphan          ConcreteResourceInstanceNodeFunc
	ConcreteResourceInstanceDeposed ConcreteResourceInstanceDeposedNodeFunc
	ConcreteModule                  ConcreteModuleNodeFunc

	// Plan Operation this graph will be used for.
	Operation walkOperation

	// ExternalReferences allows the external caller to pass in references to
	// nodes that should not be pruned even if they are not referenced within
	// the actual graph.
	ExternalReferences []*addrs.Reference

	// Overrides provides the set of overrides supplied by the testing
	// framework.
	Overrides *mocking.Overrides

	// ImportTargets are the list of resources to import.
	ImportTargets []*ImportTarget

	// forgetResources lists the resources that are to be forgotten, i.e. removed
	// from state without destroying.
	forgetResources []addrs.ConfigResource

	// forgetModules lists the modules that are to be forgotten, i.e. removed
	// from state without destroying.
	forgetModules []addrs.Module

	// GenerateConfig tells Terraform where to write and generated config for
	// any import targets that do not already have configuration.
	//
	// If empty, then config will not be generated.
	GenerateConfigPath string
}

// See GraphBuilder
func (b *PlanGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for %s", b.Operation)
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "PlanGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *PlanGraphBuilder) Steps() []GraphTransformer {
	switch b.Operation {
	case walkPlan:
		b.initPlan()
	case walkPlanDestroy:
		b.initDestroy()
	case walkValidate:
		b.initValidate()
	case walkImport:
		b.initImport()
	default:
		panic("invalid plan operation: " + b.Operation.String())
	}

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Config:   b.Config,
			destroy:  b.Operation == walkDestroy || b.Operation == walkPlanDestroy,

			importTargets: b.ImportTargets,

			// We only want to generate config during a plan operation.
			generateConfigPathForImportTargets: b.GenerateConfigPath,
		},

		// Add dynamic values
		&RootVariableTransformer{
			Config:       b.Config,
			RawValues:    b.RootVariableValues,
			Planning:     true,
			DestroyApply: false, // always false for planning
		},
		&ModuleVariableTransformer{
			Config:       b.Config,
			Planning:     true,
			DestroyApply: false, // always false for planning
		},
		&variableValidationTransformer{
			validateWalk: b.Operation == walkValidate,
		},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{
			Config:      b.Config,
			RefreshOnly: b.skipPlanChanges || b.preDestroyRefresh,
			Destroying:  b.Operation == walkPlanDestroy,
			Overrides:   b.Overrides,

			// NOTE: We currently treat anything built with the plan graph
			// builder as "planning" for our purposes here, because we share
			// the same graph node implementation between all of the walk
			// types and so the pre-planning walks still think they are
			// producing a plan even though we immediately discard it.
			Planning: true,
		},

		// Add nodes and edges for the check block assertions. Check block data
		// sources were added earlier.
		&checkTransformer{
			Config:    b.Config,
			Operation: b.Operation,
		},

		// Add orphan resources
		&OrphanResourceInstanceTransformer{
			Concrete: b.ConcreteResourceOrphan,
			State:    b.State,
			Config:   b.Config,
			skip:     b.Operation == walkPlanDestroy,
		},

		// We also need nodes for any deposed instance objects present in the
		// state, so we can plan to destroy them. (During plan this will
		// intentionally skip creating nodes for _current_ objects, since
		// ConfigTransformer created nodes that will do that during
		// DynamicExpand.)
		&StateTransformer{
			ConcreteCurrent: b.ConcreteResourceInstance,
			ConcreteDeposed: b.ConcreteResourceInstanceDeposed,
			State:           b.State,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Create orphan output nodes
		&OrphanOutputTransformer{
			Config:   b.Config,
			State:    b.State,
			Planning: true,
		},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// add providers
		transformProviders(b.ConcreteProvider, b.Config, b.ExternalProviderConfigs),

		// Remove modules no longer present in the config
		&RemovedModuleTransformer{Config: b.Config, State: b.State},

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Plugins: b.Plugins, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Concrete: b.ConcreteModule, Config: b.Config},

		// Plug in any external references.
		&ExternalReferenceTransformer{
			ExternalReferences: b.ExternalReferences,
		},

		&ReferenceTransformer{},

		&OutputReferencesTransformer{},

		&AttachDependenciesTransformer{},

		// Make sure data sources are aware of any depends_on from the
		// configuration
		&attachDataResourceDependsOnTransformer{},

		// DestroyEdgeTransformer is only required during a plan so that the
		// TargetsTransformer can determine which nodes to keep in the graph.
		&DestroyEdgeTransformer{
			Operation: b.Operation,
		},

		&pruneUnusedNodesTransformer{
			skip: b.Operation != walkPlanDestroy,
		},

		// Target
		&TargetsTransformer{Targets: b.Targets},

		// Detect when create_before_destroy must be forced on for a particular
		// node due to dependency edges, to avoid graph cycles during apply.
		&ForcedCBDTransformer{},

		// Close any ephemeral resource instances.
		&ephemeralResourceCloseTransformer{skip: b.Operation == walkValidate},

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

func (b *PlanGraphBuilder) initPlan() {
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
			preDestroyRefresh:    b.preDestroyRefresh,
			forceReplace:         b.ForceReplace,
		}
	}

	b.ConcreteResourceOrphan = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  b.skipRefresh,
			skipPlanChanges:              b.skipPlanChanges,
			forgetResources:              b.forgetResources,
			forgetModules:                b.forgetModules,
		}
	}

	b.ConcreteResourceInstanceDeposed = func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey:                   key,

			skipRefresh:     b.skipRefresh,
			skipPlanChanges: b.skipPlanChanges,
			forgetResources: b.forgetResources,
			forgetModules:   b.forgetModules,
		}
	}
}

func (b *PlanGraphBuilder) initDestroy() {
	b.initPlan()

	b.ConcreteResourceInstance = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlanDestroyableResourceInstance{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  b.skipRefresh,
		}
	}
}

func (b *PlanGraphBuilder) initValidate() {
	// Set the provider to the normal provider. This will ask for input.
	b.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	b.ConcreteResource = func(a *NodeAbstractResource) dag.Vertex {
		return &NodeValidatableResource{
			NodeAbstractResource: a,
		}
	}

	b.ConcreteModule = func(n *nodeExpandModule) dag.Vertex {
		return &nodeValidateModule{
			nodeExpandModule: *n,
		}
	}
}

func (b *PlanGraphBuilder) initImport() {
	b.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	b.ConcreteResource = func(a *NodeAbstractResource) dag.Vertex {
		return &nodeExpandPlannableResource{
			NodeAbstractResource: a,

			// For now we always skip planning changes for import, since we are
			// not going to combine importing with other changes. This is
			// temporary to try and maintain existing import behaviors, but
			// planning will need to be allowed for more complex configurations.
			skipPlanChanges: true,

			// We also skip refresh for now, since the plan output is written
			// as the new state, and users are not expecting the import process
			// to update any other instances in state.
			skipRefresh: true,
		}
	}
}
