package terraform

import (
	"log"

	"github.com/hashicorp/terraform/states"
)

// StateTransformer is a GraphTransformer that adds the elements of
// the state to the graph.
//
// This transform is used for example by the DestroyPlanGraphBuilder to ensure
// that only resources that are in the state are represented in the graph.
type StateTransformer struct {
	// ConcreteCurrent and ConcreteDeposed are used to specialize the abstract
	// resource instance nodes that this transformer will create.
	//
	// If either of these is nil, the objects of that type will be skipped and
	// not added to the graph at all. It doesn't make sense to use this
	// transformer without setting at least one of these, since that would
	// skip everything and thus be a no-op.
	ConcreteCurrent ConcreteResourceInstanceNodeFunc
	ConcreteDeposed ConcreteResourceInstanceDeposedNodeFunc

	State *states.State
}

func (t *StateTransformer) Transform(g *Graph) error {
	if !t.State.HasResources() {
		log.Printf("[TRACE] StateTransformer: state is empty, so nothing to do")
		return nil
	}

	switch {
	case t.ConcreteCurrent != nil && t.ConcreteDeposed != nil:
		log.Printf("[TRACE] StateTransformer: creating nodes for both current and deposed instance objects")
	case t.ConcreteCurrent != nil:
		log.Printf("[TRACE] StateTransformer: creating nodes for current instance objects only")
	case t.ConcreteDeposed != nil:
		log.Printf("[TRACE] StateTransformer: creating nodes for deposed instance objects only")
	default:
		log.Printf("[TRACE] StateTransformer: pointless no-op call, creating no nodes at all")
	}

	for _, ms := range t.State.Modules {
		for _, rs := range ms.Resources {
			resourceAddr := rs.Addr

			for key, is := range rs.Instances {
				addr := resourceAddr.Instance(key)

				if obj := is.Current; obj != nil && t.ConcreteCurrent != nil {
					abstract := NewNodeAbstractResourceInstance(addr)
					node := t.ConcreteCurrent(abstract)
					g.Add(node)
					log.Printf("[TRACE] StateTransformer: added %T for %s current object", node, addr)
				}

				if t.ConcreteDeposed != nil {
					for dk := range is.Deposed {
						abstract := NewNodeAbstractResourceInstance(addr)
						node := t.ConcreteDeposed(abstract, dk)
						g.Add(node)
						log.Printf("[TRACE] StateTransformer: added %T for %s deposed object %s", node, addr, dk)
					}
				}
			}
		}
	}

	return nil
}
