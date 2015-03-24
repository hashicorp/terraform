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
		targetedNodes, err := t.selectTargetedNodes(g)
		if err != nil {
			return err
		}

		for _, v := range g.Vertices() {
			if !targetedNodes.Include(v) {
				g.Remove(v)
			}
		}
	}
	return nil
}

func (t *TargetsTransformer) selectTargetedNodes(g *Graph) (*dag.Set, error) {
	targetedNodes := new(dag.Set)
	for _, v := range g.Vertices() {
		// Keep all providers; they'll be pruned later if necessary
		if r, ok := v.(GraphNodeProvider); ok {
			targetedNodes.Add(r)
			continue
		}

		// For the remaining filter, we only care about Resources and their deps
		r, ok := v.(*GraphNodeConfigResource)
		if !ok {
			continue
		}

		if t.resourceIsTarget(r) {
			targetedNodes.Add(r)

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

func (t *TargetsTransformer) resourceIsTarget(r *GraphNodeConfigResource) bool {
	for _, target := range t.Targets {
		if target == r.Name() {
			return true
		}
	}
	return false
}
