package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	SetTargets([]addrs.Targetable)
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
	Targets []addrs.Targetable

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
	if len(t.Targets) > 0 {
		targetedNodes, err := t.selectTargetedNodes(g, t.Targets)
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

// Returns a set of targeted nodes. A targeted node is either addressed
// directly, address indirectly via its container, or it's a dependency of a
// targeted node. Destroy mode keeps dependents instead of dependencies.
func (t *TargetsTransformer) selectTargetedNodes(g *Graph, addrs []addrs.Targetable) (*dag.Set, error) {
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
	return t.addDependencies(targetedNodes, g)
}

func (t *TargetsTransformer) addDependencies(targetedNodes *dag.Set, g *Graph) (*dag.Set, error) {
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

	return targetedNodes.Filter(func(dv interface{}) bool {
		return filterPartialOutputs(dv, targetedNodes, g)
	}), nil
}

// Outputs may have been included transitively, but if any of their
// dependencies have been pruned they won't be resolvable.
// If nothing depends on the output, and the output is missing any
// dependencies, remove it from the graph.
// This essentially maintains the previous behavior where interpolation in
// outputs would fail silently, but can now surface errors where the output
// is required.
func filterPartialOutputs(v interface{}, targetedNodes *dag.Set, g *Graph) bool {
	// should this just be done with TargetDownstream?
	if _, ok := v.(*NodeApplyableOutput); !ok {
		return true
	}

	dependers := g.UpEdges(v)
	for _, d := range dependers.List() {
		if _, ok := d.(*NodeCountBoundary); ok {
			continue
		}

		if !targetedNodes.Include(d) {
			// this one is going to be removed, so it doesn't count
			continue
		}

		// as soon as we see a real dependency, we mark this as
		// non-removable
		return true
	}

	depends := g.DownEdges(v)

	for _, d := range depends.List() {
		if !targetedNodes.Include(d) {
			log.Printf("[WARN] %s missing targeted dependency %s, removing from the graph",
				dag.VertexName(v), dag.VertexName(d))
			return false
		}
	}
	return true
}

func (t *TargetsTransformer) nodeIsTarget(v dag.Vertex, targets []addrs.Targetable) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case GraphNodeResourceInstance:
		vertexAddr = r.ResourceInstanceAddr()
	case GraphNodeResource:
		vertexAddr = r.ResourceAddr()
	default:
		// Only resource and resource instance nodes can be targeted.
		return false
	}
	_, ok := v.(GraphNodeResource)
	if !ok {
		return false
	}

	for _, targetAddr := range targets {
		if t.IgnoreIndices {
			// If we're ignoring indices then we'll convert any resource instance
			// addresses into resource addresses. We don't need to convert
			// vertexAddr because instance addresses are contained within
			// their associated resources, and so .TargetContains will take
			// care of this for us.
			if instance, isInstance := targetAddr.(addrs.AbsResourceInstance); isInstance {
				targetAddr = instance.ContainingResource()
			}
		}
		if targetAddr.TargetContains(vertexAddr) {
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
