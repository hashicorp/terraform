package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
)

// DeferralsTransformer is a GraphTransformer that finds the nodes in the current
// deferrals set and replaces them with a placeholder node to prevent evaluation.
// It also applies the same transformation to any dependencies of the deferred nodes.
//
// The deferral set is constructed during the Refresh and Plan walks, so only during
// the Apply walk is it guaranteed that dependencies of deferrals will not be evaluated.
type DeferralsTransformer struct {

	// The deferrals to consider when transforming the graph.
	Deferrals *Deferrals

	// Set to true when we're in a `terraform destroy` or a
	// `terraform plan -destroy`
	Destroy bool
}

func (t *DeferralsTransformer) Transform(g *Graph) error {
	deferred, err := t.selectDeferredNodes(g, t.Deferrals)
	if err != nil {
		return err
	}

	for _, nI := range deferred.List() {
		n := nI.(dag.Vertex)

		// Rather than removing the node entirely, we instead wrap it in a
		// placeholder node so that it's impotent but still visible in
		// "terraform graph".
		log.Printf("[TRACE] Suppressing deferred node %q\n", dag.VertexName(n))

		placeholder := &graphNodeDeferred{n}
		g.Replace(n, placeholder)
	}

	return nil
}

func (t *DeferralsTransformer) selectDeferredNodes(g *Graph, deferrals *Deferrals) (*dag.Set, error) {
	deferred := new(dag.Set)

	modDeferrals := deferrals.ModuleByPath(g.Path)
	if modDeferrals == nil {
		// No deferrals for this module.
		return deferred, nil
	}

	addToSet := func(node dag.Vertex) error {
		deferred.Add(node)

		var deps *dag.Set
		var err error
		if t.Destroy {
			// Walk direction is reversed when destroying
			deps, err = g.Ancestors(node)
		} else {
			deps, err = g.Descendents(node)
		}
		if err != nil {
			return err
		}

		for _, d := range deps.List() {
			deferred.Add(d)
		}

		return nil
	}

	for _, v := range g.Vertices() {
		if pn, ok := v.(GraphNodeProvider); ok {
			if modDeferrals.ProviderIsDeferred(pn.ProviderName()) {
				err := addToSet(v)
				if err != nil {
					return nil, err
				}
			}
		}
		// TODO: Resource deferrals.
		// This might require a new node interface to get the module-local resource
		// name for us to use to match.
	}

	return deferred, nil
}

type graphNodeDeferred struct {
	DeferredNode dag.Vertex
}

func (n *graphNodeDeferred) Name() string {
	return dag.VertexName(n.DeferredNode)
}

func (n *graphNodeDeferred) DotOrigin() bool {
	if nd, ok := n.DeferredNode.(GraphNodeDotOrigin); ok {
		return nd.DotOrigin()
	} else {
		return false
	}
}

func (n *graphNodeDeferred) DotNode(name string, opts *GraphDotOpts) *dot.Node {
	var node *dot.Node

	if nd, ok := n.DeferredNode.(GraphNodeDotter); ok {
		node = nd.DotNode(name, opts)
	} else {
		return nil
	}

	// Deferred nodes appear in grey in the graph
	node.Attrs["color"] = "gray"
	node.Attrs["fontcolor"] = "gray"

	return node
}
