// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// InitGraphBuilder is a GraphBuilder implementation that builds a graph for
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
type InitGraphBuilder struct {
	// Config is the configuration tree to build a plan from.
	Config *configs.Config

	// RootVariableValues are the raw input values for root input variables
	// given by the caller, which we'll resolve into final values as part
	// of the plan walk.
	RootVariableValues InputValues

	ConcreteProvider                ConcreteProviderNodeFunc
	ConcreteResource                ConcreteResourceNodeFunc
	ConcreteResourceInstance        ConcreteResourceInstanceNodeFunc
	ConcreteResourceOrphan          ConcreteResourceInstanceNodeFunc
	ConcreteResourceInstanceDeposed ConcreteResourceInstanceDeposedNodeFunc
	ConcreteModule                  ConcreteModuleNodeFunc
}

// See GraphBuilder
func (b *InitGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for init")
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "InitGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *InitGraphBuilder) Steps() []GraphTransformer {
	// Copy from initPlan
	b.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	b.ConcreteResource = func(a *NodeAbstractResource) dag.Vertex {
		return &nodeExpandPlannableResource{
			NodeAbstractResource: a,
		}
	}

	b.ConcreteResourceOrphan = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
		}
	}

	b.ConcreteResourceInstanceDeposed = func(a *NodeAbstractResourceInstance, key states.DeposedKey) dag.Vertex {
		return &NodePlanDeposedResourceInstanceObject{
			NodeAbstractResourceInstance: a,
			DeposedKey:                   key,
		}
	}

	steps := []GraphTransformer{
		// Creates all the resources represented in the config
		&ConfigTransformer{
			Concrete: b.ConcreteResource,
			Config:   b.Config,
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
		&variableValidationTransformer{},
		&LocalTransformer{Config: b.Config},

		// TODO: Add transformer for backend blocks
		// TODO: Add transformer for provider requirements
		// TODO: Add transformer for module requirements
		// TODO: Add transformer for edges

		// At some point: Add resources back into the graph? We would need state which is only available after the backend has been evaluated so maybe we need actually two graphs, one to get the backend and therefore the state and then one to get all the resources.
	}

	return steps
}
