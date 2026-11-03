// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodePlanDestroyRefreshResourceInstance refreshes a current managed resource
// instance object from state during a destroy plan. It is referenceable by
// the resource address, so that provider configurations, data sources, and
// other values that must be evaluated during a destroy see the refreshed
// object. The corresponding NodePlanDestroyableResourceInstance depends on
// this node, and plans the destroy from the refreshed object.
//
// This node only ever participates in forward (evaluation) ordering, and
// nothing in the forward ordering may depend on a destroy node, so combining
// these with the destroy edges cannot introduce cycles.
type nodePlanDestroyRefreshResourceInstance struct {
	*NodeAbstractResourceInstance

	skipRefresh bool

	// destroyDependencies are the resources which destroy-time evaluation
	// for this instance depends on in the current configuration, such as
	// through its provider configuration. These are attached via the
	// corresponding destroy node, and merged into the object in the refresh
	// state, whether or not it is refreshed, so that the apply-time destroy
	// order allows that evaluation.
	destroyDependencies []addrs.ConfigResource

	// result is the upgraded and refreshed object, which the dependent destroy
	// node consumes. The graph edge between the two orders these accesses.
	result *states.ResourceInstanceObject
}

var (
	_ GraphNodeModuleInstance       = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeExecutable           = (*nodePlanDestroyRefreshResourceInstance)(nil)
	_ GraphNodeProviderConsumer     = (*nodePlanDestroyRefreshResourceInstance)(nil)
)

func (n *nodePlanDestroyRefreshResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String() + " (refresh)"
}

// References returns nothing, because refreshing only uses the prior state
// and does not evaluate the resource configuration.
func (n *nodePlanDestroyRefreshResourceInstance) References() []*addrs.Reference {
	return nil
}

func (n *nodePlanDestroyRefreshResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	state, _, readDiags := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Record any schema upgrades in the previous run state, so that it
	// conforms to the current schema.
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, prevRunState))
	if diags.HasErrors() {
		return diags
	}

	obj := state
	if !n.skipRefresh {
		refreshed, deferred, refreshDiags := n.refresh(ctx, states.NotDeposed, state, ctx.Deferrals().DeferralAllowed())
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}
		if deferred != nil {
			// Leave the prior object in place, the destroy node will report
			// the deferral.
			log.Printf("[TRACE] nodePlanDestroyRefreshResourceInstance: refresh of %s deferred", addr)
		} else {
			obj = refreshed
		}
	}

	// The apply-time destroy order must allow destroy-time evaluation, so the
	// dependencies through the current configuration are recorded whether or
	// not the object was refreshed.
	if obj != nil {
		obj.Dependencies = mergeDeps(n.destroyDependencies, obj.Dependencies)
	}
	n.result = obj

	diags = diags.Append(n.writeResourceInstanceState(ctx, obj, refreshState))
	if diags.HasErrors() {
		return diags
	}
	return diags.Append(n.writeResourceInstanceState(ctx, obj, workingState))
}

// destroyPlanRefreshTransformer adds a nodePlanDestroyRefreshResourceInstance
// for every current managed resource instance object in state, and makes the
// corresponding destroy node depend on it.
type destroyPlanRefreshTransformer struct {
	State       *states.State
	skipRefresh bool
	skip        bool
}

func (t *destroyPlanRefreshTransformer) Transform(g *Graph) error {
	if t.skip || t.State == nil {
		return nil
	}

	destroyers := addrs.MakeMap[addrs.AbsResourceInstance, *NodePlanDestroyableResourceInstance]()
	for v := range g.VerticesSeq() {
		if d, ok := v.(*NodePlanDestroyableResourceInstance); ok {
			destroyers.Put(d.ResourceInstanceAddr(), d)
		}
	}

	for _, ms := range t.State.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			for key, is := range rs.Instances {
				if is.Current == nil {
					continue
				}
				addr := rs.Addr.Instance(key)
				node := &nodePlanDestroyRefreshResourceInstance{
					NodeAbstractResourceInstance: NewNodeAbstractResourceInstance(addr),
					skipRefresh:                  t.skipRefresh,
				}
				g.Add(node)
				if d, ok := destroyers.GetOk(addr); ok {
					d.refresh = node
					g.Connect(d, node)
				}
			}
		}
	}
	return nil
}
