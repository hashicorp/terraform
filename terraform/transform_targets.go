package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// TargetsTransformer is a GraphTransformer that, when the user specifies a
// list of resources to target, limits the graph to only those resources and
// their dependencies.
type TargetsTransformer struct {
	// List of targeted resource names specified by the user
	Targets []string

	// Set to true when we're in a `terraform destroy` or a
	// `terraform plan -destroy`
	Destroy bool
}

func (t *TargetsTransformer) Transform(g *Graph) error {
	if len(t.Targets) > 0 {
		// TODO: duplicated in OrphanTransformer; pull up parsing earlier
		addrs, err := t.parseTargetAddresses()
		if err != nil {
			return err
		}

		targetedNodes, err := t.selectTargetedNodes(g, addrs)
		if err != nil {
			return err
		}

		for _, v := range g.Vertices() {
			if _, ok := v.(GraphNodeAddressable); ok {
				if !targetedNodes.Include(v) {
					log.Printf("[DEBUG] Removing %q, filtered by targeting.", dag.VertexName(v))
					g.Remove(v)
				}
			}
		}
	}
	return nil
}

func (t *TargetsTransformer) parseTargetAddresses() ([]ResourceAddress, error) {
	addrs := make([]ResourceAddress, len(t.Targets))
	for i, target := range t.Targets {
		ta, err := ParseResourceAddress(target)
		if err != nil {
			return nil, err
		}
		addrs[i] = *ta
	}
	return addrs, nil
}

// Returns the list of targeted nodes. A targeted node is either addressed
// directly, or is an Ancestor of a targeted node. Destroy mode keeps
// Descendents instead of Ancestors.
func (t *TargetsTransformer) selectTargetedNodes(
	g *Graph, addrs []ResourceAddress) (*dag.Set, error) {
	targetedNodes := new(dag.Set)
	for _, v := range g.Vertices() {
		if t.nodeIsTarget(v, addrs) {
			targetedNodes.Add(v)

			// We inform nodes that ask about the list of targets - helps for nodes
			// that need to dynamically expand. Note that this only occurs for nodes
			// that are already directly targeted.
			if tn, ok := v.(GraphNodeTargetable); ok {
				tn.SetTargets(addrs)
			}

			var deps *dag.Set
			var err error
			if t.Destroy {
				deps, err = g.Descendents(v)
			} else {
				deps, err = g.Ancestors(v)
			}
			if err != nil {
				return nil, err
			}

			for _, d := range deps.List() {
				targetedNodes.Add(d)
			}
		}
	}
	return targetedNodes, nil
}

func (t *TargetsTransformer) nodeIsTarget(
	v dag.Vertex, addrs []ResourceAddress) bool {
	r, ok := v.(GraphNodeAddressable)
	if !ok {
		return false
	}
	addr := r.ResourceAddress()
	for _, targetAddr := range addrs {
		if targetAddr.Equals(addr) {
			return true
		}
	}
	return false
}
