// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
)

// GraphNodeDestroyer must be implemented by nodes that destroy resources.
type GraphNodeDestroyer interface {
	dag.Vertex

	// DestroyAddr is the address of the resource that is being
	// destroyed by this node. If this returns nil, then this node
	// is not destroying anything.
	DestroyAddr() *addrs.AbsResourceInstance
}

// GraphNodeCreator must be implemented by nodes that create OR update resources.
type GraphNodeCreator interface {
	// CreateAddr is the address of the resource being created or updated
	CreateAddr() *addrs.AbsResourceInstance
}

// DestroyEdgeTransformer is a GraphTransformer that creates the proper
// references for destroy resources. Destroy resources are more complex
// in that they must be depend on the destruction of resources that
// in turn depend on the CREATION of the node being destroy.
//
// That is complicated. Visually:
//
//	B_d -> A_d -> A -> B
//
// Notice that A destroy depends on B destroy, while B create depends on
// A create. They're inverted. This must be done for example because often
// dependent resources will block parent resources from deleting. Concrete
// example: VPC with subnets, the VPC can't be deleted while there are
// still subnets.
type DestroyEdgeTransformer struct {
	// FIXME: GraphNodeCreators are not always applying changes, and should not
	// participate in the destroy graph if there are no operations which could
	// interract with destroy nodes. We need Changes for now to detect the
	// action type, but perhaps this should be indicated somehow by the
	// DiffTransformer which was intended to be the only transformer operating
	// from the change set.
	Changes *plans.ChangesSrc

	// FIXME: Operation will not be needed here one we can better track
	// inter-provider dependencies and remove the cycle checks in
	// tryInterProviderDestroyEdge.
	Operation walkOperation
}

// tryInterProviderDestroyEdge checks if we're inserting a destroy edge
// across a provider boundary, and only adds the edge if it results in no cycles.
//
// FIXME: The cycles can arise in valid configurations when a provider depends
// on resources from another provider. In the future we may want to inspect
// the dependencies of the providers themselves, to avoid needing to use the
// blunt hammer of checking for cycles.
//
// A reduced example of this dependency problem looks something like:
/*

createA <-               createB
  |        \            /    |
  |         providerB <-     |
  v                     \    v
destroyA ------------->  destroyB

*/
//
// The edge from destroyA to destroyB would be skipped in this case, but there
// are still other combinations of changes which could connect the A and B
// groups around providerB in various ways.
//
// The most difficult problem here happens during a full destroy operation.
// That creates a special case where resources on which a provider depends must
// exist for evaluation before they are destroyed. This means that any provider
// dependencies must wait until all that provider's resources have first been
// destroyed. This is where these cross-provider edges are still required to
// ensure the correct order.
func (t *DestroyEdgeTransformer) tryInterProviderDestroyEdge(g *Graph, from, to dag.Vertex) {
	e := dag.BasicEdge(from, to)
	g.Connect(e)

	// If this is a complete destroy operation, then there are no create/update
	// nodes to worry about and we can accept the edge without deeper inspection.
	if t.Operation == walkDestroy || t.Operation == walkPlanDestroy {
		return
	}

	// getComparableProvider inspects the node to try and get the most precise
	// description of the provider being used to help determine if 2 nodes are
	// from the same provider instance.
	getComparableProvider := func(pc GraphNodeProviderConsumer) string {
		ps := pc.Provider().String()

		// we don't care about `exact` here, since we're only looking for any
		// clue that the providers may differ.
		p, _ := pc.ProvidedBy()
		switch p := p.(type) {
		case addrs.AbsProviderConfig:
			ps = p.String()
		case addrs.LocalProviderConfig:
			ps = p.String()
		}

		return ps
	}

	pc, ok := from.(GraphNodeProviderConsumer)
	if !ok {
		return
	}
	fromProvider := getComparableProvider(pc)

	pc, ok = to.(GraphNodeProviderConsumer)
	if !ok {
		return
	}
	toProvider := getComparableProvider(pc)

	// Check for cycles, and back out the edge if there are any.
	// The cycles we are looking for only appears between providers, so don't
	// waste time checking for cycles if both nodes use the same provider.
	if fromProvider != toProvider && g.Ancestors(to).Include(from) {
		log.Printf("[DEBUG] DestroyEdgeTransformer: skipping inter-provider edge %s->%s which creates a cycle",
			dag.VertexName(from), dag.VertexName(to))
		g.RemoveEdge(e)
	}
}

func (t *DestroyEdgeTransformer) Transform(g *Graph) error {
	// Build a map of what is being destroyed (by address string) to
	// the list of destroyers.
	destroyers := make(map[string][]GraphNodeDestroyer)

	// Record the creators, which will need to depend on the destroyers if they
	// are only being updated.
	creators := make(map[string][]GraphNodeCreator)

	// destroyersByResource records each destroyer by the ConfigResource
	// address.  We use this because dependencies are only referenced as
	// resources and have no index or module instance information, but we will
	// want to connect all the individual instances for correct ordering.
	destroyersByResource := make(map[string][]GraphNodeDestroyer)
	for _, v := range g.Vertices() {
		switch n := v.(type) {
		case GraphNodeDestroyer:
			addrP := n.DestroyAddr()
			if addrP == nil {
				log.Printf("[WARN] DestroyEdgeTransformer: %q (%T) has no destroy address", dag.VertexName(n), v)
				continue
			}
			addr := *addrP

			key := addr.String()
			log.Printf("[TRACE] DestroyEdgeTransformer: %q (%T) destroys %s", dag.VertexName(n), v, key)
			destroyers[key] = append(destroyers[key], n)

			resAddr := addr.ContainingResource().Config().String()
			destroyersByResource[resAddr] = append(destroyersByResource[resAddr], n)
		case GraphNodeCreator:
			addr := n.CreateAddr()
			cfgAddr := addr.ContainingResource().Config().String()

			if t.Changes == nil {
				// unit tests may not have changes
				creators[cfgAddr] = append(creators[cfgAddr], n)
				break
			}

			// NoOp changes should not participate in the destroy dependencies.
			rc := t.Changes.ResourceInstance(*addr)
			if rc != nil && rc.Action != plans.NoOp {
				creators[cfgAddr] = append(creators[cfgAddr], n)
			}
		}
	}

	// If we aren't destroying anything, there will be no edges to make
	// so just exit early and avoid future work.
	if len(destroyers) == 0 {
		return nil
	}

	// Go through and connect creators to destroyers. Going along with
	// our example, this makes: A_d => A
	for _, v := range g.Vertices() {
		cn, ok := v.(GraphNodeCreator)
		if !ok {
			continue
		}

		addr := cn.CreateAddr()
		if addr == nil {
			continue
		}

		for _, d := range destroyers[addr.String()] {
			// For illustrating our example
			a_d := d.(dag.Vertex)
			a := v

			log.Printf(
				"[TRACE] DestroyEdgeTransformer: connecting creator %q with destroyer %q",
				dag.VertexName(a), dag.VertexName(a_d))

			g.Connect(dag.BasicEdge(a, a_d))
		}
	}

	// connect creators to any destroyers on which they may depend
	for _, cs := range creators {
		for _, c := range cs {
			ri, ok := c.(GraphNodeResourceInstance)
			if !ok {
				continue
			}

			for _, resAddr := range ri.StateDependencies() {
				for _, desDep := range destroyersByResource[resAddr.String()] {
					if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(c, desDep) {
						log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(c), dag.VertexName(desDep))
						g.Connect(dag.BasicEdge(c, desDep))
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(c), dag.VertexName(desDep))
					}
				}
			}
		}
	}

	// Connect destroy dependencies as stored in the state
	for _, ds := range destroyers {
		for _, des := range ds {
			ri, ok := des.(GraphNodeResourceInstance)
			if !ok {
				continue
			}

			for _, resAddr := range ri.StateDependencies() {
				for _, desDep := range destroyersByResource[resAddr.String()] {
					if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(desDep, des) {
						log.Printf("[TRACE] DestroyEdgeTransformer: %s has stored dependency of %s\n", dag.VertexName(desDep), dag.VertexName(des))
						t.tryInterProviderDestroyEdge(g, desDep, des)
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(desDep), dag.VertexName(des))
					}
				}

				// We can have some create or update nodes which were
				// dependents of the destroy node. If they have no destroyer
				// themselves, make the connection directly from the creator.
				for _, createDep := range creators[resAddr.String()] {
					if !graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(createDep, des) {
						log.Printf("[DEBUG] DestroyEdgeTransformer2: %s has stored dependency of %s\n", dag.VertexName(createDep), dag.VertexName(des))
						t.tryInterProviderDestroyEdge(g, createDep, des)
					} else {
						log.Printf("[TRACE] DestroyEdgeTransformer2: skipping %s => %s inter-module-instance dependency\n", dag.VertexName(createDep), dag.VertexName(des))
					}
				}
			}
		}
	}

	return nil
}

// Remove nodes that aren't needed, when planning or applying a full destroy.
// Specifically, we want to remove any temporary values (variables, outputs,
// locals) that aren't ultimately referenced by a provider, as well as any
// expanders whose instances are never relevant. This is necessary because of
// some interacting behaviors:
//
// - In a destroy, we create nodes from the full config, but then try to *act*
// like we're using a config with all the resources removed. So we have many
// nodes for temporary values that *would not exist* in the stipulated
// "resources gone" config. (This is why we *can* prune some nodes.)
//
// - We still need provider configurations to destroy anything. Provider configs
// must be re-evaluated during apply (they aren't cached in the state or plan),
// and they might refer to temporary values like locals or variables. (This is
// why we can't just prune *all* the temporary value nodes.)
//
// - Any node referenced by an in-use provider should end up properly anchored
// in the destroy graph ordering. But other temporary values are more randomly
// ordered (because we don't bother fixing edges to guarantee they happen before
// relevant destructions), so they might be impossible to evaluate properly
// during a destroy, especially if we already performed a partial destroy and
// got interrupted. (This is why we *must* prune some nodes.)
//
// - The first and third points above aren't relevant in a normal run that
// happens to perform some destructions, because any temporary value that
// references a destroyed resource will get ordered after the creation of the
// resource's replacement. (This is why we only prune for destroys.)
type pruneUnusedNodesTransformer struct {
	// Both the plan and apply graph builders will skip this transformer except
	// during a full destroy.
	skip bool
}

func (t *pruneUnusedNodesTransformer) Transform(g *Graph) error {
	if t.skip {
		return nil
	}

	// we need to track nodes to keep, because the dependency trees can overlap,
	// so we can't just remove all dependencies of nodes we don't want.
	keep := make(dag.Set)

	// Only keep destroyers, their providers, and anything the providers need
	// for configuration. Since the destroyer should already be hooked up to the
	// provider, keeping all the destroyer dependencies should suffice.
	for _, n := range g.Vertices() {
		// a special case of destroyer, is that by convention Terraform expects
		// root outputs to be "destroyed", and the output node is what writes
		// the nil state. A root module output currently identifies itself as a
		// temporary value which is not temporary for that reason.
		if tmp, ok := n.(graphNodeTemporaryValue); ok && !tmp.temporaryValue() {
			log.Printf("[TRACE] pruneUnusedNodesTransformer: keeping root output %s", dag.VertexName(n))
			keep.Add(n)
			continue
		}

		// from here we only search for managed resource destroy nodes
		n, ok := n.(GraphNodeDestroyer)
		if !ok {
			continue
		}

		log.Printf("[TRACE] pruneUnusedNodesTransformer: keeping destroy node %s", dag.VertexName(n))
		keep.Add(n)

		for _, anc := range g.Ancestors(n) {
			log.Printf("[TRACE] pruneUnusedNodesTransformer: keeping %s as dependency of %s", dag.VertexName(anc), dag.VertexName(n))
			keep.Add(anc)
		}
	}

	for _, n := range g.Vertices() {
		if !keep.Include(n) {
			log.Printf("[TRACE] pruneUnusedNodesTransformer: removing %s", dag.VertexName(n))
			g.Remove(n)
		}
	}

	return nil
}
