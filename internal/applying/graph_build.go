package applying

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/internal/resources"
	"github.com/hashicorp/terraform/internal/schemas"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

func buildGraph(
	priorState *states.State,
	config *configs.Config,
	plan *plans.Plan,
	schemas *schemas.Schemas,
) (*dag.AcyclicGraph, tfdiags.Diagnostics) {
	graph := &dag.AcyclicGraph{}
	var diags tfdiags.Diagnostics
	const errorSummary = "Failed to construct graph for terraform apply"

	// 🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔
	// Currently our plan structure throws away a lot of context we learned
	// during the plan walk, so sadly we need to do a bunch of work here
	// to recreate that context by inferring things from the configuration
	// and state. In future it would be nice to make the plan format a more
	// direct representation of the graph of actions and their dependencies
	// so that this function could load it directly into the graph, and
	// then we'd use the configuration only to find the expressions that we
	// need to re-evaluate during the apply walk in order to complete our
	// planned values.
	// 🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔🤔

	resourceActions, err := buildGraphResourceActions(graph, priorState, config, plan, schemas)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errorSummary,
			fmt.Sprintf("Error while analyzing resource changes: %s.\n\nThis is a bug in Terraform; please report it.", err),
		))
	}

	_, err = buildProviderConfigActions(graph, resourceActions, config, schemas)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errorSummary,
			fmt.Sprintf("Error while analyzing provider configurations: %s.\n\nThis is a bug in Terraform; please report it.", err),
		))
	}

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
//
// This function is responsible for creating all of the non-reference-derived
// dependency edges between the actions it creates, with the exception of
// edges with provider configurations that must be handled separately by
// the caller.
func buildGraphResourceActions(
	g *dag.AcyclicGraph,
	priorState *states.State,
	config *configs.Config,
	plan *plans.Plan,
	schemas *schemas.Schemas,
) (map[string]*resourceActions, error) {
	actions := make(map[string]*resourceActions)

	// The planned changes are the main decider for what actions we'll create,
	// but we'll also refer to the prior state and plan for additional
	// context about dependencies because currently the plan representation
	// is not sufficient on its own.
	for _, ricSrc := range plan.Changes.Resources {
		instanceAddr := ricSrc.Addr
		resourceAddr := instanceAddr.ContainingResource()
		resourceMapKey := resourceAddr.String()
		resourceSchema, _ := schemas.ResourceTypeConfig(
			ricSrc.ProviderAddr.ProviderConfig.Type.LegacyString(),
			resourceAddr.Resource.Mode,
			resourceAddr.Resource.Type,
		)
		ric, err := ricSrc.Decode(resourceSchema.ImpliedType())
		if err != nil {
			return nil, fmt.Errorf("invalid plan for %s: %s", instanceAddr, tfdiags.FormatError(err))
		}
		var resourceConfig *configs.Resource
		if modCfg := config.DescendentForInstance(resourceAddr.Module); modCfg != nil {
			resourceConfig = modCfg.Module.ResourceByAddr(resourceAddr.Resource)
			// Note that resourceConfig might still be nil, because it's
			// valid to have destroy changes for instances belonging to
			// resources that are no longer in the configuration.
		}

		if _, exists := actions[resourceMapKey]; !exists {
			var deps []addrs.Referenceable
			if resourceConfig != nil {
				configDeps := resources.ResourceDependencies(resourceConfig, resourceSchema, schemas.Provisioners)
				deps = append(deps, configDeps...)
			}

			actions[resourceMapKey] = &resourceActions{
				Addr:              resourceAddr,
				Instances:         make(map[addrs.InstanceKey]*resourceInstanceActions),
				ProviderConfigRef: ric.ProviderAddr,
				Dependencies:      deps,
			}
		}

		rActions := actions[resourceMapKey]
		instanceKey := instanceAddr.Resource.Key
		if _, exists := rActions.Instances[instanceKey]; !exists {
			rActions.Instances[instanceAddr.Resource.Key] = &resourceInstanceActions{
				Addr:           instanceAddr,
				DestroyDeposed: make(map[states.DeposedKey]*resourceInstanceDestroyChangeAction),
			}
		}

		riActions := rActions.Instances[instanceKey]
		needCreateUpdate := ric.Action != plans.Delete
		needDestroy := ric.Action == plans.Delete || ric.Action.IsReplace()

		// First we'll deal with actions for the resource as a whole. Since
		// whole resources are not tracked in the plan, we're using the
		// individual instances to hint which resource-level actions we need,
		// so here we're lazy-populating actions the first time we visit an
		// instance that gives us the appropriate hint.
		if needCreateUpdate && rActions.SetMeta == nil {
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
		if needDestroy {
			// If we have at least one delete action (where replace actions
			// imply a delete) then we potentially need an action to tidy
			// up the leftover resource shell in the state if the resource
			// is no longer present in the configuration and there are no
			// instances left after all of the other operations.
			if rActions.Cleanup == nil {
				needCleanup := false
				if modCfg := config.DescendentForInstance(resourceAddr.Module); modCfg == nil {
					needCleanup = true
				} else if resourceCfg := modCfg.Module.ResourceByAddr(resourceAddr.Resource); resourceCfg == nil {
					needCleanup = true
				}
				if needCleanup {
					action := &resourceCleanupAction{
						Addr: resourceAddr,
					}
					rActions.Cleanup = action
					g.Add(action)
				}
			}
		}

		// Now we'll deal with the resource instance itself. We will produce
		// either one or two actions here, because "replace" actions in the
		// plan are really just shorthand for separate create and destroy
		// actions in a particular order.
		if needCreateUpdate {
			if ric.DeposedKey != "" {
				// This should never happen: the only valid action for
				// a deposed object is to destroy it.
				panic(fmt.Sprintf("plan intends to %s a deposed object of %s", ric.Action, instanceAddr))
			}
			if resourceConfig == nil {
				// Configuration can be absent only for destroy.
				panic(fmt.Sprintf("plan intends to %s an instance of resource %s that is not in the configuration", ric.Action, resourceAddr))
			}
			actionType := ric.Action
			if actionType.IsReplace() {
				// A replace action lowers into separate create/destroy
				// actions.
				actionType = plans.Create
			}
			action := &resourceInstanceNonDestroyChangeAction{
				Addr:          instanceAddr,
				Action:        actionType,
				Config:        resourceConfig,
				PriorObj:      ric.Before,
				PlannedNewObj: ric.After,
			}
			riActions.CreateUpdate = action
			g.Add(action)
		}
		if needDestroy {
			actionType := ric.Action
			if actionType.IsReplace() {
				// A replace action lowers into separate create/destroy
				// actions.
				actionType = plans.Delete
			}
			action := &resourceInstanceDestroyChangeAction{
				Addr:       instanceAddr,
				DeposedKey: ric.DeposedKey,
				Action:     actionType,
				PriorObj:   ric.Before,
			}
			if ric.DeposedKey == states.NotDeposed {
				riActions.Destroy = action
			} else {
				riActions.DestroyDeposed[ric.DeposedKey] = action
			}
			g.Add(action)
		}
		if ric.Action.IsReplace() {
			// When we're replacing we have two nodes, which need a dependency
			// edge between them to select the correct ordering.
			switch ric.Action {
			case plans.CreateThenDelete:
				g.Connect(dag.BasicEdge(riActions.Destroy, riActions.CreateUpdate))
			case plans.DeleteThenCreate:
				g.Connect(dag.BasicEdge(riActions.CreateUpdate, riActions.Destroy))
			}
		}
	}

	for _, rActions := range actions {
		if rActions.SetMeta != nil {
			// All of the instance actions must happen after the metadata
			// has been set.
			for _, riActions := range rActions.Instances {
				if riActions.CreateUpdate != nil {
					g.Connect(dag.BasicEdge(riActions.CreateUpdate, rActions.SetMeta))
				}
				if riActions.Destroy != nil {
					g.Connect(dag.BasicEdge(riActions.Destroy, rActions.SetMeta))
				}
				for _, deposedAction := range riActions.DestroyDeposed {
					g.Connect(dag.BasicEdge(deposedAction, rActions.SetMeta))
				}
			}
		}
		if rActions.Cleanup != nil {
			// Cleanup must happen after all other actions related to the
			// resource.
			for _, riActions := range rActions.Instances {
				if riActions.CreateUpdate != nil {
					g.Connect(dag.BasicEdge(rActions.Cleanup, riActions.CreateUpdate))
				}
				if riActions.Destroy != nil {
					g.Connect(dag.BasicEdge(rActions.Cleanup, riActions.Destroy))
				}
				for _, deposedAction := range riActions.DestroyDeposed {
					g.Connect(dag.BasicEdge(rActions.Cleanup, deposedAction))
				}
			}
		}
		if rActions.SetMeta != nil && rActions.Cleanup != nil {
			// Cleanup must also happen after SetMeta. This edge is usually
			// redundant given the connection with the resource instance
			// actions we created above, but we'll insert it to ensure
			// completeness/correctness anyway and then let the caller run
			// transitive reduction to detect if this really is redundant.
			g.Connect(dag.BasicEdge(rActions.Cleanup, rActions.SetMeta))
		}
	}

	return actions, nil
}

func buildProviderConfigActions(
	g *dag.AcyclicGraph,
	resourceActions map[string]*resourceActions,
	config *configs.Config,
	schemas *schemas.Schemas,
) (map[string]*providerConfigActions, error) {
	actions := make(map[string]*providerConfigActions)

	// We use our resource actions as the primary driver for creating provider
	// configuration actions here because that way we will include only the
	// minimal set of provider configurations we need for this particular
	// plan, without needing to delete any nodes/edges after the fact.
	for _, rActions := range resourceActions {
		providerConfigAddr := rActions.ProviderConfigRef
		providerConfigKey := providerConfigAddr.String()
		var providerConfig *configs.Provider
		if modCfg := config.DescendentForInstance(providerConfigAddr.Module); modCfg != nil {
			providerConfig = modCfg.Module.ProviderConfigs[providerConfigAddr.ProviderConfig.Type.LegacyString()]
		}
		// Note that providerConfig can still be nil here, because Terraform
		// permits omitting a root module provider configuration block
		// entirely if it would otherwise have been empty anyway.

		// We'll lazy create the actions for a provider config the first
		// time we see it, and then just connect it to any subsequent
		// resource actions that refer to it.
		if _, exists := actions[providerConfigKey]; !exists {
			initAction := &instantiateProviderAction{
				Addr:   providerConfigAddr,
				Config: providerConfig,
			}
			closeAction := &closeProviderAction{
				Addr:   providerConfigAddr,
				Config: providerConfig,
			}
			actions[providerConfigKey] = &providerConfigActions{
				Instantiate: initAction,
				Close:       closeAction,
			}
			g.Add(initAction)
			g.Add(closeAction)
		}

		rActions.AllRequire(actions[providerConfigKey].Instantiate, g)
		rActions.AllRequiredBy(actions[providerConfigKey].Close, g)
	}

	return actions, nil
}
