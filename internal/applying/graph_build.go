package applying

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

func buildGraph(
	priorState *states.State,
	config *configs.Config,
	plan *plans.Plan,
) (*dag.AcyclicGraph, tfdiags.Diagnostics) {
	graph := &dag.AcyclicGraph{}
	var diags tfdiags.Diagnostics

	// TODO: Later, when we're talking other actions below, we can use
	// the result of this to create the necessary dependency edges.
	_ = buildGraphResourceActions(graph, priorState, config, plan)

	// Remove as many edges as we can while retaining correctness of edges
	// overall. For example, if a -> c and a -> b -> c then we can remove
	// a -> c safely; it's implied by a -> b -> c.
	graph.TransitiveReduction()

	return graph, diags
}

// buildGraphResourceActions inserts into the graph the action nodes for
// all of the resources and resource instances with planned changes,
// returning a map describing the action nodes it created and the addresses
// of objects whose actions each one depends on.
func buildGraphResourceActions(
	g *dag.AcyclicGraph,
	priorState *states.State,
	config *configs.Config,
	plan *plans.Plan,
) map[string]*resourceActions {
	actions := make(map[string]*resourceActions)

	// The planned changes are the main decider for what actions we'll create,
	// but we'll also refer to the prior state and plan for additional
	// context about dependencies because currently the plan representation
	// is not sufficient on its own.
	for _, ric := range plan.Changes.Resources {
		instanceAddr := ric.Addr
		resourceAddr := instanceAddr.ContainingResource()
		resourceMapKey := resourceAddr.String()

		if _, exists := actions[resourceMapKey]; !exists {
			var deps []addrs.Referenceable
			if modCfg := config.DescendentForInstance(resourceAddr.Module); modCfg != nil {
				if resourceCfg := modCfg.Module.ResourceByAddr(resourceAddr.Resource); resourceCfg != nil {
					// TODO: Analyze the configuration for dependencies.
				}
			}

			actions[resourceMapKey] = &resourceActions{
				Addr:              resourceAddr,
				Instances:         make(map[addrs.InstanceKey]resourceInstanceActions),
				ProviderConfigRef: ric.ProviderAddr,
				Dependencies:      deps,
			}
		}

		rActions := actions[resourceMapKey]

		if ric.Action != plans.Delete && rActions.SetMeta == nil {
			// If we have at least one non-delete action then we need an
			// action to set the meta information for the whole resource.
			action := &resourceSetMetaAction{
				Addr:           resourceAddr,
				ProviderConfig: rActions.ProviderConfigRef,

				// We use whatever instance we find first as an example for
				// the instance key, because our plan format doesn't currently
				// record this explicitly. This is safe because we're only
				// doing this for non-delete actions and any instances keys
				// not consistent with the new each mode would've been planned
				// for deletion.
				EachMode: states.EachModeForInstanceKey(instanceAddr.Resource.Key),
			}
			rActions.SetMeta = action
			g.Add(action)
		}

		if ric.Action == plans.Delete || ric.Action.IsReplace() {
			// If we have at least one delete action (where replace actions
			// imply a delete) then we need an action to potentially tidy
			// up the leftover resource "husk" if there are no instances left
			// after all of the delete operations.

			// TODO: Implement this
		}

		// TODO: Decode the plan and populate all of the instance action fields
		// properly. But to do that we'll need the provider schemas.

	}

	return actions
}
