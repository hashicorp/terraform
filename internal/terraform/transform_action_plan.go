// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ActionPlanTransformer struct {
	Skip           bool
	Config         *configs.Config
	Targets        []addrs.Targetable
	Operation      walkOperation
	ConcreteAction ConcreteActionNodeFunc
}

func (t *ActionPlanTransformer) Transform(g *Graph) error {
	if t.Skip {
		return nil
	}

	if t.Operation != walkPlan {
		return nil
	}

	// First add all action config nodes
	err := t.transformActionConfig(g, t.Config)
	if err != nil {
		return err
	}

	if len(t.Targets) > 0 {
		// Then we're invoking and we're just going to include the actions that
		// have been specifically asked for.

		for _, target := range t.Targets {
			var config *configs.Action
			switch target := target.(type) {
			case addrs.AbsAction:
				module := t.Config.DescendantForInstance(target.Module)
				if module != nil {
					config = module.Module.Actions[target.Action.String()]
				}
			case addrs.AbsActionInstance:
				module := t.Config.DescendantForInstance(target.Module)
				if module != nil {
					config = module.Module.Actions[target.Action.Action.String()]
				}
			}

			if config == nil {
				return fmt.Errorf("action %s does not exist in the configuration", target.String())
			}

			g.Add(&nodeActionInvokeExpand{
				Target: target,
				Config: config,
			})
		}

		return nil
	}

	// otherwise, add all the action triggers from the config.

	return t.transformActionTrigger(g, t.Config)
}

func (t *ActionPlanTransformer) transformActionConfig(g *Graph, config *configs.Config) error {
	// Add our actions
	if err := t.transformActionConfigSingle(g, config); err != nil {
		return err
	}

	// Transform all the children.
	for _, c := range config.Children {
		if err := t.transformActionConfig(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ActionPlanTransformer) transformActionConfigSingle(g *Graph, config *configs.Config) error {
	// collect all the Action Declarations (configs.Actions) in this module so
	// we can validate that actions referenced in a resource's ActionTriggers
	// exist in this module.
	allConfigActions := make(map[string]*configs.Action)
	for _, a := range config.Module.Actions {
		if a != nil {
			addr := a.Addr().InModule(config.Path)
			allConfigActions[addr.String()] = a
			log.Printf("[TRACE] ConfigTransformer: Adding action %s", addr)
			abstract := &NodeAbstractAction{
				Addr:   addr,
				Config: *a,
			}
			var node dag.Vertex
			if f := t.ConcreteAction; f != nil {
				node = f(abstract)
			} else {
				node = DefaultConcreteActionNodeFunc(abstract)
			}
			g.Add(node)
		}
	}

	var diags tfdiags.Diagnostics
	for _, r := range config.Module.ManagedResources {
		// Verify that any actions referenced in the resource's ActionTriggers exist in this module
		if r.Managed != nil && r.Managed.ActionTriggers != nil {
			for i, at := range r.Managed.ActionTriggers {
				for _, action := range at.Actions {

					refs, parseRefDiags := langrefs.ReferencesInExpr(addrs.ParseRef, action.Expr)
					if parseRefDiags != nil {
						return parseRefDiags.Err()
					}

					var configAction addrs.ConfigAction

					for _, ref := range refs {
						switch a := ref.Subject.(type) {
						case addrs.Action:
							configAction = a.InModule(config.Path)
						case addrs.ActionInstance:
							configAction = a.Action.InModule(config.Path)
						case addrs.CountAttr, addrs.ForEachAttr:
							// nothing to do, these will get evaluated later
						default:
							// This should have been caught during validation
							panic(fmt.Sprintf("unexpected action address %T", a))
						}
					}

					_, ok := allConfigActions[configAction.String()]
					if !ok {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Configuration for triggered action does not exist",
							Detail:   fmt.Sprintf("The configuration for the given action %s does not exist. All triggered actions must have an associated configuration.", configAction.String()),
							Subject:  &r.Managed.ActionTriggers[i].DeclRange,
						})
					}
				}
			}
		}
	}
	if diags.HasErrors() {
		return diags.Err()
	}

	return nil
}

func (t *ActionPlanTransformer) transformActionTrigger(g *Graph, config *configs.Config) error {
	// Add our action triggers
	if err := t.transformActionTriggerSingle(g, config); err != nil {
		return err
	}

	// Transform all the children.
	for _, c := range config.Children {
		if err := t.transformActionTrigger(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ActionPlanTransformer) transformActionTriggerSingle(g *Graph, config *configs.Config) error {
	actionConfigs := addrs.MakeMap[addrs.ConfigAction, *configs.Action]()
	for _, a := range config.Module.Actions {
		actionConfigs.Put(a.Addr().InModule(config.Path), a)
	}

	resourceNodes := addrs.MakeMap[addrs.ConfigResource, []GraphNodeConfigResource]()
	for _, node := range g.Vertices() {
		rn, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}
		// We ignore any instances that _also_ implement
		// GraphNodeResourceInstance, since in the unlikely event that they
		// do exist we'd probably end up creating cycles by connecting them.
		if _, ok := node.(GraphNodeResourceInstance); ok {
			continue
		}

		rAddr := rn.ResourceAddr()
		resourceNodes.Put(rAddr, append(resourceNodes.Get(rAddr), rn))
	}

	for _, r := range config.Module.ManagedResources {
		priorNodes := []*nodeActionTriggerPlanExpand{}
		for i, at := range r.Managed.ActionTriggers {
			for j, action := range at.Actions {
				refs, parseRefDiags := langrefs.ReferencesInExpr(addrs.ParseRef, action.Expr)
				if parseRefDiags != nil {
					return parseRefDiags.Err()
				}

				var configAction addrs.ConfigAction

				for _, ref := range refs {
					switch a := ref.Subject.(type) {
					case addrs.Action:
						configAction = a.InModule(config.Path)
					case addrs.ActionInstance:
						configAction = a.Action.InModule(config.Path)
					case addrs.CountAttr, addrs.ForEachAttr:
						// nothing to do, these will get evaluated later
					default:
						// This should have been caught during validation
						panic(fmt.Sprintf("unexpected action address %T", a))
					}
				}

				actionConfig, ok := actionConfigs.GetOk(configAction)
				if !ok {
					// This should have been caught during validation
					panic(fmt.Sprintf("action config not found for %s", configAction))
				}

				resourceAddr := r.Addr().InModule(config.Path)
				resourceNode, ok := resourceNodes.GetOk(resourceAddr)
				if !ok {
					panic(fmt.Sprintf("Could not find node for %s", resourceAddr))
				}

				nat := &nodeActionTriggerPlanExpand{
					Addr:   configAction,
					Config: actionConfig,
					lifecycleActionTrigger: &lifecycleActionTrigger{
						events:                  at.Events,
						resourceAddress:         resourceAddr,
						actionExpr:              action.Expr,
						actionTriggerBlockIndex: i,
						actionListIndex:         j,
						invokingSubject:         action.Expr.Range().Ptr(),
						conditionExpr:           at.Condition,
					},
				}

				g.Add(nat)

				// We always want to plan after the resource is done planning
				for _, node := range resourceNode {
					g.Connect(dag.BasicEdge(nat, node))
				}

				// We want to plan after all prior nodes
				for _, priorNode := range priorNodes {
					g.Connect(dag.BasicEdge(nat, priorNode))
				}
				priorNodes = append(priorNodes, nat)
			}
		}
	}

	return nil
}
