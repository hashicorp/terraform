package terraform

import "github.com/hashicorp/terraform/dag"

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
			if targetedNodes.Include(v) {
			} else {
				g.Remove(v)
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

func (t *TargetsTransformer) selectTargetedNodes(
	g *Graph, addrs []ResourceAddress) (*dag.Set, error) {
	targetedNodes := new(dag.Set)
	for _, v := range g.Vertices() {
		// Keep all providers; they'll be pruned later if necessary
		if r, ok := v.(GraphNodeProvider); ok {
			targetedNodes.Add(r)
			continue
		}

		// For the remaining filter, we only care about addressable nodes
		r, ok := v.(GraphNodeAddressable)
		if !ok {
			continue
		}

		if t.nodeIsTarget(r, addrs) {
			targetedNodes.Add(r)
			// If the node would like to know about targets, tell it.
			if n, ok := r.(GraphNodeTargetable); ok {
				n.SetTargets(addrs)
			}

			var deps *dag.Set
			var err error
			if t.Destroy {
				deps, err = g.Descendents(r)
			} else {
				deps, err = g.Ancestors(r)
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
	r GraphNodeAddressable, addrs []ResourceAddress) bool {
	addr := r.ResourceAddress()
	for _, targetAddr := range addrs {
		if targetAddr.Equals(addr) {
			return true
		}
	}
	return false
}
