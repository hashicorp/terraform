package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// DiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type DiffTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc
	State    *states.State
	Changes  *plans.Changes
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
	resourceNodes := map[string][]GraphNodeResource{}
	for _, node := range g.Vertices() {
		rn, ok := node.(GraphNodeResource)
		if !ok {
			continue
		}
		// We ignore any instances that _also_ implement
		// GraphNodeResourceInstance, since in the unlikely event that they
		// do exist we'd probably end up creating cycles by connecting them.
		if _, ok := node.(GraphNodeResourceInstance); ok {
			continue
		}

		addr := rn.ResourceAddr().String()
		resourceNodes[addr] = append(resourceNodes[addr], rn)
	}

	for _, rc := range changes.Resources {
		addr := rc.Addr
		dk := rc.DeposedKey

		log.Printf("[TRACE] DiffTransformer: found %s change for %s %s", rc.Action, addr, dk)

		// Depending on the action we'll need some different combinations of
		// nodes, because destroying uses a special node type separate from
		// other actions.
		var update, delete, createBeforeDestroy bool
		switch rc.Action {
		case plans.NoOp:
			continue
		case plans.Delete:
			delete = true
		case plans.DeleteThenCreate, plans.CreateThenDelete:
			update = true
			delete = true
			createBeforeDestroy = (rc.Action == plans.CreateThenDelete)
		default:
			update = true
		}

		if dk != states.NotDeposed && update {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid planned change for deposed object",
				fmt.Sprintf("The plan contains a non-delete change for %s deposed object %s. The only valid action for a deposed object is to destroy it, so this is a bug in Terraform.", addr, dk),
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

			g.Add(node)
			rsrcAddr := addr.ContainingResource().String()
			for _, rsrcNode := range resourceNodes[rsrcAddr] {
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
					DeposedKey:                   dk,
				}
				node.(*NodeDestroyResourceInstance).ModifyCreateBeforeDestroy(createBeforeDestroy)
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
			rsrcAddr := addr.ContainingResource().String()
			for _, rsrcNode := range resourceNodes[rsrcAddr] {
				// We connect this edge "forwards" (even though destroy dependencies
				// are often inverted) because evaluating the resource node
				// after the destroy node could cause an unnecessary husk of
				// a resource state to be re-added.
				g.Connect(dag.BasicEdge(node, rsrcNode))
			}
		}

	}

	log.Printf("[TRACE] DiffTransformer complete")

	return diags.Err()
}
