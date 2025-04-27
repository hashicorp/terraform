// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// walkOperation is an enum which tells the walkContext what to do.
type queryWalkOperation byte

// TODO: Generate stringer for this enum
const (
	queryWalkInvalid queryWalkOperation = iota
	queryWalkValidate
	queryWalkEval
)

// QueryGraphBuilder is a GraphBuilder implementation that builds a graph for
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
type QueryGraphBuilder struct {
	// Config is the configuration tree to build a plan from.
	Config *configs.Config

	Operation queryWalkOperation

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

	// GenerateConfig tells Terraform where to write and generated config for
	// any import targets that do not already have configuration.
	//
	// If empty, then config will not be generated.
	GenerateConfigPath string
}

// See GraphBuilder
func (b *QueryGraphBuilder) Build() (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building query graph for %v", b.Operation)
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "QueryGraphBuilder",
	}).Build(addrs.RootModuleInstance)
}

// See GraphBuilder
func (b *QueryGraphBuilder) Steps() []GraphTransformer {
	var concreteProvider ConcreteProviderNodeFunc = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the lists represented in the config
		&QueryConfigTransformer{
			Config:                             b.Config,
			generateConfigPathForImportTargets: b.GenerateConfigPath,
		},

		// Add dynamic values
		&RootVariableTransformer{
			Config:       b.Config,
			RawValues:    b.RootVariableValues,
			Planning:     false,
			DestroyApply: false,
		},
		&variableValidationTransformer{
			validateWalk: b.Operation == queryWalkValidate,
		},
		&LocalTransformer{Config: b.Config},

		// add providers
		transformProviders(concreteProvider, b.Config, b.ExternalProviderConfigs),

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		DynamicTransformer(func(g *Graph) error {
			for _, v := range g.Vertices() {
				nq, ok := v.(*NodeQueryList)
				if !ok {
					continue
				}
				if tv, ok := v.(AttachListSchema); ok {
					providerFqn := tv.Provider()

					// Resource schema
					schema, err := b.Plugins.ResourceTypeSchema(providerFqn, addrs.ManagedResourceMode, nq.Addr().Type)
					if err != nil {
						return fmt.Errorf("failed to read schema for %s in %s: %s", nq.Addr(), providerFqn, err)
					}
					if schema.Body == nil {
						log.Printf("[ERROR] AttachListSchema: No resource schema available for %s", nq.Addr())
						continue
					}
					log.Printf("[TRACE] AttachListSchema: attaching resource schema to %s", dag.VertexName(v))
					tv.AttachResourceSchema(&schema)

					// List schema
					schema, err = b.Plugins.ResourceTypeSchema(providerFqn, addrs.ListResourceMode, nq.Addr().Type)
					if err != nil {
						return fmt.Errorf("failed to read schema for %s in %s: %s", nq.Addr(), providerFqn, err)
					}
					if schema.Body == nil {
						log.Printf("[ERROR] AttachListSchema: No list resource schema available for %s", nq.Addr())
						continue
					}
					log.Printf("[TRACE] AttachListSchema: attaching list resource schema to %s", dag.VertexName(v))
					tv.AttachSchema(&schema)
				}
			}
			return nil
		}),

		&ReferenceTransformer{},

		// Target
		&TargetsTransformer{Targets: b.Targets},

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
