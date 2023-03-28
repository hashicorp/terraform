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

	// AutoApprovedPlan is true if we are executing a plan, and it has been told
	// to auto approve.
	//
	// The graph will skip executing checks during an auto approved plan, since
	// the checks will be redone during the apply stage and reporting them
	// during the plan just pollutes the output since the user can't respond
	// to anything the checks say anyway (since the plan has been preapproved).
	AutoApprovedPlan bool
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
					raiseChecks:   !t.AutoApprovedPlan,
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

			// This part ensures we report our checks before our nested data
			// block executes and attempts to report on a check.
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
						continue
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
		return true
	default:
		// For everything else, we still want to validate the checks make sense
		// logically and syntactically, but we won't actually resolve the check
		// conditions.
		return false
	}
}

// RaiseChecks returns true if this operation should report the results of the
// checks as well as executing them.
//
// In practice, we don't show the check results during an auto approved plan
// simply because the apply operation will happen immediately and also report
// more relevant check results.
func (t *checkTransformer) RaiseChecks() bool {
	return !t.AutoApprovedPlan
}
