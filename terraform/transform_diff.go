package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// DiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type DiffTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc
	Changes  *plans.Changes
}

func (t *DiffTransformer) Transform(g *Graph) error {
	if t.Changes == nil || len(t.Changes.Resources) == 0 {
		// Nothing to do!
		return nil
	}

	// Go through all the modules in the diff.
	log.Printf("[TRACE] DiffTransformer starting")

	for _, rc := range t.Changes.Resources {
		addr := rc.Addr
		dk := rc.DeposedKey

		// Depending on the action we'll need some different combinations of
		// nodes, because destroying uses a special node type separate from
		// other actions.
		var update, delete bool
		switch rc.Action {
		case plans.NoOp:
			continue
		case plans.Delete:
			delete = true
		case plans.Replace:
			update = true
			delete = true
		default:
			update = true
		}

		if update {
			// All actions except destroying the node type chosen by t.Concrete
			abstract := NewNodeAbstractResourceInstance(addr)
			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}

			if dk != states.NotDeposed {
				// The only valid action for deposed objects is to destroy them.
				// Entering this branch suggests a bug in the plan phase that
				// proposed this change.
				return fmt.Errorf("invalid %s action for deposed object on %s: only Delete is allowed", rc.Action, addr)
			}

			log.Printf("[TRACE] DiffTransformer: %s will be represented by %s", addr, dag.VertexName(node))
			g.Add(node)
		}

		if delete {
			// Destroying always uses this destroy-specific node type.
			abstract := NewNodeAbstractResourceInstance(addr)
			node := &NodeDestroyResourceInstance{
				NodeAbstractResourceInstance: abstract,
				DeposedKey:                   dk,
			}
			if dk == states.NotDeposed {
				log.Printf("[TRACE] DiffTransformer: %s will be represented for destruction by %s", addr, dag.VertexName(node))
			} else {
				log.Printf("[TRACE] DiffTransformer: %s deposed object %s will be represented for destruction by %s", addr, dk, dag.VertexName(node))
			}
			g.Add(node)
		}

	}

	log.Printf("[TRACE] DiffTransformer complete")

	return nil
}
