// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// DiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type DiffTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc
	State    *states.State
	Changes  *plans.ChangesSrc
	Config   *configs.Config
}

// return true if the given resource instance has either Preconditions or
// Postconditions defined in the configuration.
func (t *DiffTransformer) hasConfigConditions(addr addrs.AbsResourceInstance) bool {
	// unit tests may have no config
	if t.Config == nil {
		return false
	}

	cfg := t.Config.DescendantForInstance(addr.Module)
	if cfg == nil {
		return false
	}

	res := cfg.Module.ResourceByAddr(addr.ConfigResource().Resource)
	if res == nil {
		return false
	}

	return len(res.Preconditions) > 0 || len(res.Postconditions) > 0
}

func (t *DiffTransformer) Transform(g *Graph) error {
	if t.Changes == nil || len(t.Changes.Resources) == 0 {
		// Nothing to do!
		return nil
	}

	// Go through all the modules in the diff.
	log.Printf("[TRACE] DiffTransformer starting")

	var diags tfdiags.Diagnostics
	state := t.State
	changes := t.Changes

	// DiffTransformer creates resource _instance_ nodes. If there are any
	// whole-resource nodes already in the graph, we must ensure that they
	// get evaluated before any of the corresponding instances by creating
	// dependency edges, so we'll do some prep work here to ensure we'll only
	// create connections to nodes that existed before we started here.
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

	// We will partition the action invocations into two groups based on if they are supposed to
	// run before or after the resource change.
	// We want to attach before-triggered action invocations to the triggering resource instance
	// to be run as part of the apply phase.
	// The after-triggered action invocations will be run as part of a separate node
	// that will be connected to the resource instance nodes.
	runBeforeNode := addrs.MakeMap[addrs.AbsResourceInstance, []*plans.ActionInvocationInstanceSrc]()
	runAfterNode := addrs.MakeMap[addrs.AbsResourceInstance, []*plans.ActionInvocationInstanceSrc]()
	var cliTriggeredNode []*plans.ActionInvocationInstanceSrc
	for _, ai := range changes.ActionInvocations {
		if _, ok := ai.ActionTrigger.(plans.InvokeCmdActionTrigger); ok {
			cliTriggeredNode = append(cliTriggeredNode, ai)
			continue
		}
		lat := ai.ActionTrigger.(plans.LifecycleActionTrigger)

		var targetMap addrs.Map[addrs.AbsResourceInstance, []*plans.ActionInvocationInstanceSrc]

		switch lat.TriggerEvent() {
		case configs.BeforeCreate, configs.BeforeUpdate, configs.BeforeDestroy:
			targetMap = runBeforeNode
		case configs.AfterCreate, configs.AfterUpdate, configs.AfterDestroy:
			targetMap = runAfterNode
		default:
			panic("I don't know when to run this action invocation")
		}

		basis := []*plans.ActionInvocationInstanceSrc{}
		if targetMap.Has(lat.TriggeringResourceAddr) {
			basis = targetMap.Get(lat.TriggeringResourceAddr)
		}

		targetMap.Put(lat.TriggeringResourceAddr, append(basis, ai))
	}

	for _, rc := range changes.Resources {
		addr := rc.Addr
		dk := rc.DeposedKey

		log.Printf("[TRACE] DiffTransformer: found %s change for %s %s", rc.Action, addr, dk)

		// Depending on the action we'll need some different combinations of
		// nodes, because destroying uses a special node type separate from
		// other actions.
		var update, delete, forget, createBeforeDestroy bool
		switch rc.Action {
		case plans.NoOp:
			// For a no-op change we don't take any action but we still
			// run any condition checks associated with the object, to
			// make sure that they still hold when considering the
			// results of other changes.
			update = t.hasConfigConditions(addr)
		case plans.Delete:
			delete = true
		case plans.DeleteThenCreate, plans.CreateThenDelete:
			update = true
			delete = true
			createBeforeDestroy = (rc.Action == plans.CreateThenDelete)
		case plans.Forget:
			forget = true
		default:
			update = true
		}

		// A deposed instance may only have a change of Delete, Forget, or NoOp.
		// A NoOp can happen if the provider shows it no longer exists during
		// the most recent ReadResource operation.
		if dk != states.NotDeposed && !(rc.Action == plans.Delete || rc.Action == plans.Forget || rc.Action == plans.NoOp) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid planned change for deposed object",
				fmt.Sprintf("The plan contains a non-remove change for %s deposed object %s. The only valid actions for a deposed object are to destroy it or remove it from state, so this is a bug in Terraform.", addr, dk),
			))
			continue
		}

		// If we're going to do a create_before_destroy Replace operation then
		// we need to allocate a DeposedKey to use to retain the
		// not-yet-destroyed prior object, so that the delete node can destroy
		// _that_ rather than the newly-created node, which will be current
		// by the time the delete node is visited.
		if update && delete && createBeforeDestroy {
			// In this case, variable dk will be the _pre-assigned_ DeposedKey
			// that must be used if the update graph node deposes the current
			// instance, which will then align with the same key we pass
			// into the destroy node to ensure we destroy exactly the deposed
			// object we expect.
			if state != nil {
				ris := state.ResourceInstance(addr)
				if ris == nil {
					// Should never happen, since we don't plan to replace an
					// instance that doesn't exist yet.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid planned change",
						fmt.Sprintf("The plan contains a replace change for %s, which doesn't exist yet. This is a bug in Terraform.", addr),
					))
					continue
				}

				// Allocating a deposed key separately from using it can be racy
				// in general, but we assume here that nothing except the apply
				// node we instantiate below will actually make new deposed objects
				// in practice, and so the set of already-used keys will not change
				// between now and then.
				dk = ris.FindUnusedDeposedKey()
			} else {
				// If we have no state at all yet then we can use _any_
				// DeposedKey.
				dk = states.NewDeposedKey()
			}
		}

		if update {
			// All actions except destroying the node type chosen by t.Concrete
			abstract := NewNodeAbstractResourceInstance(addr)
			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}

			if createBeforeDestroy {
				// We'll attach our pre-allocated DeposedKey to the node if
				// it supports that. NodeApplyableResourceInstance is the
				// specific concrete node type we are looking for here really,
				// since that's the only node type that might depose objects.
				if dn, ok := node.(GraphNodeDeposer); ok {
					dn.SetPreallocatedDeposedKey(dk)
				}
				log.Printf("[TRACE] DiffTransformer: %s will be represented by %s, deposing prior object to %s", addr, dag.VertexName(node), dk)
			} else {
				log.Printf("[TRACE] DiffTransformer: %s will be represented by %s", addr, dag.VertexName(node))
			}

			// We only need to attach actions to updating nodes for now
			// (until before_destroy & after destroy are added)
			if beforeActions, ok := runBeforeNode.GetOk(addr); ok {
				if attachBeforeActionsNode, ok := node.(*NodeApplyableResourceInstance); ok {
					attachBeforeActionsNode.beforeActionInvocations = beforeActions
				}
			}

			g.Add(node)
			for _, rsrcNode := range resourceNodes.Get(addr.ConfigResource()) {
				g.Connect(dag.BasicEdge(node, rsrcNode))
			}
		}

		if delete {
			// Destroying always uses a destroy-specific node type, though
			// which one depends on whether we're destroying a current object
			// or a deposed object.
			var node GraphNodeResourceInstance
			abstract := NewNodeAbstractResourceInstance(addr)
			if dk == states.NotDeposed {
				node = &NodeDestroyResourceInstance{
					NodeAbstractResourceInstance: abstract,
				}
			} else {
				node = &NodeDestroyDeposedResourceInstanceObject{
					NodeAbstractResourceInstance: abstract,
					DeposedKey:                   dk,
				}
			}
			if dk == states.NotDeposed {
				log.Printf("[TRACE] DiffTransformer: %s will be represented for destruction by %s", addr, dag.VertexName(node))
			} else {
				log.Printf("[TRACE] DiffTransformer: %s deposed object %s will be represented for destruction by %s", addr, dk, dag.VertexName(node))
			}
			g.Add(node)
		}

		if forget {
			var node GraphNodeResourceInstance
			abstract := NewNodeAbstractResourceInstance(addr)
			if dk == states.NotDeposed {
				node = &NodeForgetResourceInstance{
					NodeAbstractResourceInstance: abstract,
				}
			} else {
				node = &NodeForgetDeposedResourceInstanceObject{
					NodeAbstractResourceInstance: abstract,
					DeposedKey:                   dk,
				}
			}

			log.Printf("[TRACE] DiffTransformer: %s will be represented for forgetting by %s", addr, dag.VertexName(node))
			g.Add(node)
		}

	}

	// Create a node for each resource instance that invokes all the action invocations that are
	// supposed to run after the resource change.
	for key, value := range runAfterNode.Iter() {
		if len(value) == 0 {
			continue
		}

		log.Printf("[TRACE] DiffTransformer: adding action invocations to run after %s", key)
		actionNode := &nodeActionApply{
			TriggeringResourceaddrs: &key,
			ActionInvocations:       value,
		}

		// Find the config resource associated with this. While for each resource instance all
		// actions need to run in sequence, for different resource instances they can run in
		// parallel.
		resourceNode, ok := resourceNodes.GetOk(key.ConfigResource())
		if !ok {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Missing resource node for action invocations",
				fmt.Sprintf("Could not find resource node for action invocations for %s", key),
			))
			continue
		}

		g.Add(actionNode)
		for _, rNode := range resourceNode {
			g.Connect(dag.BasicEdge(actionNode, rNode))
		}
	}

	for _, value := range cliTriggeredNode {
		actionNode := &nodeActionApply{
			TriggeringResourceaddrs: nil,
			ActionInvocations:       []*plans.ActionInvocationInstanceSrc{value},
		}
		g.Add(actionNode)
	}

	log.Printf("[TRACE] DiffTransformer complete")

	return diags.Err()
}
