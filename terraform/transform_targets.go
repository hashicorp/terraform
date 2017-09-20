package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	SetTargets([]ResourceAddress)
}

// GraphNodeTargetDownstream is an interface for graph nodes that need to
// be remain present under targeting if any of their dependencies are targeted.
// TargetDownstream is called with the set of vertices that are direct
// dependencies for the node, and it should return true if the node must remain
// in the graph in support of those dependencies.
//
// This is used in situations where the dependency edges are representing an
// ordering relationship but the dependency must still be visited if its
// dependencies are visited. This is true for outputs, for example, since
// they must get updated if any of their dependent resources get updated,
// which would not normally be true if one of their dependencies were targeted.
type GraphNodeTargetDownstream interface {
	TargetDownstream(targeted, untargeted *dag.Set) bool
}

// TargetsTransformer is a GraphTransformer that, when the user specifies a
// list of resources to target, limits the graph to only those resources and
// their dependencies.
type TargetsTransformer struct {
	// List of targeted resource names specified by the user
	Targets []string

	// List of parsed targets, provided by callers like ResourceCountTransform
	// that already have the targets parsed
	ParsedTargets []ResourceAddress

	// If set, the index portions of resource addresses will be ignored
	// for comparison. This is used when transforming a graph where
	// counted resources have not yet been expanded, since otherwise
	// the unexpanded nodes (which never have indices) would not match.
	IgnoreIndices bool

	// Set to true when we're in a `terraform destroy` or a
	// `terraform plan -destroy`
	Destroy bool
}

func (t *TargetsTransformer) Transform(g *Graph) error {
	if len(t.Targets) > 0 && len(t.ParsedTargets) == 0 {
		addrs, err := t.parseTargetAddresses()
		if err != nil {
			return err
		}

		t.ParsedTargets = addrs
	}

	if len(t.ParsedTargets) > 0 {
		targetedNodes, err := t.selectTargetedNodes(g, t.ParsedTargets)
		if err != nil {
			return err
		}

		for _, v := range g.Vertices() {
			removable := false
			if _, ok := v.(GraphNodeResource); ok {
				removable = true
			}
			if vr, ok := v.(RemovableIfNotTargeted); ok {
				removable = vr.RemoveIfNotTargeted()
			}
			if removable && !targetedNodes.Include(v) {
				log.Printf("[DEBUG] Removing %q, filtered by targeting.", dag.VertexName(v))
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

// Returns the list of targeted nodes. A targeted node is either addressed
// directly, or is an Ancestor of a targeted node. Destroy mode keeps
// Descendents instead of Ancestors.
func (t *TargetsTransformer) selectTargetedNodes(
	g *Graph, addrs []ResourceAddress) (*dag.Set, error) {
	targetedNodes := new(dag.Set)

	vertices := g.Vertices()

	for _, v := range vertices {
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

	// Handle nodes that need to be included if their dependencies are included.
	// This requires multiple passes since we need to catch transitive
	// dependencies if and only if they are via other nodes that also
	// support TargetDownstream. For example:
	// output -> output -> targeted-resource: both outputs need to be targeted
	// output -> non-targeted-resource -> targeted-resource: output not targeted
	//
	// We'll keep looping until we stop targeting more nodes.
	queue := targetedNodes.List()
	for len(queue) > 0 {
		vertices := queue
		queue = nil // ready to append for next iteration if neccessary
		for _, v := range vertices {
			dependers := g.UpEdges(v)
			if dependers == nil {
				// indicates that there are no up edges for this node, so
				// we have nothing to do here.
				continue
			}

			dependers = dependers.Filter(func(dv interface{}) bool {
				// Can ignore nodes that are already targeted
				/*if targetedNodes.Include(dv) {
					return false
				}*/

				_, ok := dv.(GraphNodeTargetDownstream)
				return ok
			})

			if dependers.Len() == 0 {
				continue
			}

			for _, dv := range dependers.List() {
				if targetedNodes.Include(dv) {
					// Already present, so nothing to do
					continue
				}

				// We'll give the node some information about what it's
				// depending on in case that informs its decision about whether
				// it is safe to be targeted.
				deps := g.DownEdges(v)
				depsTargeted := deps.Intersection(targetedNodes)
				depsUntargeted := deps.Difference(depsTargeted)

				if dv.(GraphNodeTargetDownstream).TargetDownstream(depsTargeted, depsUntargeted) {
					targetedNodes.Add(dv)
					// Need to visit this node on the next pass to see if it
					// has any transitive dependers.
					queue = append(queue, dv)
				}
			}
		}
	}

	return targetedNodes, nil
}

func (t *TargetsTransformer) nodeIsTarget(
	v dag.Vertex, addrs []ResourceAddress) bool {
	r, ok := v.(GraphNodeResource)
	if !ok {
		return false
	}

	addr := r.ResourceAddr()
	for _, targetAddr := range addrs {
		if t.IgnoreIndices {
			// targetAddr is not a pointer, so we can safely mutate it without
			// interfering with references elsewhere.
			targetAddr.Index = -1
		}
		if targetAddr.Contains(addr) {
			return true
		}
	}

	return false
}

// RemovableIfNotTargeted is a special interface for graph nodes that
// aren't directly addressable, but need to be removed from the graph when they
// are not targeted. (Nodes that are not directly targeted end up in the set of
// targeted nodes because something that _is_ targeted depends on them.) The
// initial use case for this interface is GraphNodeConfigVariable, which was
// having trouble interpolating for module variables in targeted scenarios that
// filtered out the resource node being referenced.
type RemovableIfNotTargeted interface {
	RemoveIfNotTargeted() bool
}
