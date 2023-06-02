// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

type checkTransformer struct {
	// Config for the entire module.
	Config *configs.Config

	// Operation is the current operation this node will be part of.
	Operation walkOperation
}

var _ GraphTransformer = (*checkTransformer)(nil)

func (t *checkTransformer) Transform(graph *Graph) error {
	return t.transform(graph, t.Config, graph.Vertices())
}

func (t *checkTransformer) transform(g *Graph, cfg *configs.Config, allNodes []dag.Vertex) error {

	if t.Operation == walkDestroy || t.Operation == walkPlanDestroy {
		// Don't include anything about checks during destroy operations.
		//
		// For other plan and normal apply operations we do everything, for
		// destroy operations we do nothing. For any other operations we still
		// include the check nodes, but we don't actually execute the checks
		// instead we still validate their references and make sure their
		// conditions make sense etc.
		return nil
	}

	moduleAddr := cfg.Path

	for _, check := range cfg.Module.Checks {
		configAddr := check.Addr().InModule(moduleAddr)

		// We want to create a node for each check block. This node will execute
		// after anything it references, and will update the checks object
		// embedded in the plan and/or state.

		log.Printf("[TRACE] checkTransformer: Nodes and edges for %s", configAddr)
		expand := &nodeExpandCheck{
			addr:   configAddr,
			config: check,
			makeInstance: func(addr addrs.AbsCheck, cfg *configs.Check) dag.Vertex {
				return &nodeCheckAssert{
					addr:          addr,
					config:        cfg,
					executeChecks: t.ExecuteChecks(),
				}
			},
		}
		g.Add(expand)

		// We also need to report the checks we are going to execute before we
		// try and execute them.
		if t.ReportChecks() {
			report := &nodeReportCheck{
				addr: configAddr,
			}
			g.Add(report)

			// Make sure we report our checks before we start executing the
			// actual checks.
			g.Connect(dag.BasicEdge(expand, report))

			if check.DataResource != nil {
				// If we have a nested data source, we need to make sure we
				// also report the check before the data source executes.
				//
				// We loop through all the nodes in the graph to find the one
				// that contains our data source and connect it.
				for _, other := range allNodes {
					if resource, isResource := other.(GraphNodeConfigResource); isResource {
						resourceAddr := resource.ResourceAddr()
						if !resourceAddr.Module.Equal(moduleAddr) {
							// This resource isn't in the same module as our check
							// so skip it.
							continue
						}

						resourceCfg := cfg.Module.ResourceByAddr(resourceAddr.Resource)
						if resourceCfg != nil && resourceCfg.Container != nil && resourceCfg.Container.Accessible(check.Addr()) {
							// Make sure we report our checks before we execute any
							// embedded data resource.
							g.Connect(dag.BasicEdge(other, report))

							// There's at most one embedded data source, and
							// we've found it so stop looking.
							break
						}
					}
				}
			}
		}
	}

	for _, child := range cfg.Children {
		if err := t.transform(g, child, allNodes); err != nil {
			return err
		}
	}

	return nil
}

// ReportChecks returns true if this operation should report any check blocks
// that it is about to execute.
//
// This is generally only true for planning operations, as apply operations
// recreate the expected checks from the plan.
func (t *checkTransformer) ReportChecks() bool {
	return t.Operation == walkPlan
}

// ExecuteChecks returns true if this operation should actually execute any
// check blocks in the config.
//
// If this returns false we will still create and execute check nodes in the
// graph, but they will only validate things like references and syntax.
func (t *checkTransformer) ExecuteChecks() bool {
	switch t.Operation {
	case walkPlan, walkApply:
		// We only actually execute the checks for plan and apply operations.
		return true
	default:
		// For everything else, we still want to validate the checks make sense
		// logically and syntactically, but we won't actually resolve the check
		// conditions.
		return false
	}
}
