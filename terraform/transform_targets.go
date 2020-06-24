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
	TargetDownstream(targeted, untargeted dag.Set) bool
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
}

func (t *TargetsTransformer) Transform(g *Graph) error {
	if len(t.Targets) > 0 {
		targetedNodes, err := t.selectTargetedNodes(g, t.Targets)
		if err != nil {
			return err
		}

		for _, v := range g.Vertices() {
			if !targetedNodes.Include(v) {
				log.Printf("[DEBUG] Removing %q, filtered by targeting.", dag.VertexName(v))
				g.Remove(v)
			}
		}
	}

	return nil
}

// Returns a set of targeted nodes. A targeted node is either addressed
// directly, address indirectly via its container, or it's a dependency of a
// targeted node.
func (t *TargetsTransformer) selectTargetedNodes(g *Graph, addrs []addrs.Targetable) (dag.Set, error) {
	targetedNodes := make(dag.Set)

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

			deps, _ := g.Ancestors(v)
			for _, d := range deps {
				targetedNodes.Add(d)
			}
		}
	}

	// It is expected that outputs which are only derived from targeted
	// resources are also updated. While we don't include any other possible
	// side effects from the targeted nodes, these are added because outputs
	// cannot be targeted on their own.
	// Start by finding the root module output nodes themselves
	for _, v := range vertices {
		// outputs are all temporary value types
		tv, ok := v.(graphNodeTemporaryValue)
		if !ok {
			continue
		}

		// root module outputs indicate that while they are an output type,
		// they not temporary and will return false here.
		if tv.temporaryValue() {
			continue
		}

		// If this output is descended only from targeted resources, then we
		// will keep it
		deps, _ := g.Ancestors(v)
		found := 0
		for _, d := range deps {
			switch d.(type) {
			case GraphNodeResourceInstance:
			case GraphNodeConfigResource:
			default:
				continue
			}

			if !targetedNodes.Include(d) {
				// this dependency isn't being targeted, so we can't process this
				// output
				found = 0
				break
			}

			found++
		}

		if found > 0 {
			// we found an output we can keep; add it, and all it's dependencies
			targetedNodes.Add(v)
			for _, d := range deps {
				targetedNodes.Add(d)
			}
		}
	}

	return targetedNodes, nil
}

func (t *TargetsTransformer) nodeIsTarget(v dag.Vertex, targets []addrs.Targetable) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case GraphNodeResourceInstance:
		vertexAddr = r.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		vertexAddr = r.ResourceAddr()
	default:
		// Only resource and resource instance nodes can be targeted.
		return false
	}

	for _, targetAddr := range targets {
		if t.IgnoreIndices {
			// If we're ignoring indices then we'll convert any resource instance
			// addresses into resource addresses. We don't need to convert
			// vertexAddr because instance addresses are contained within
			// their associated resources, and so .TargetContains will take
			// care of this for us.
			switch instance := targetAddr.(type) {
			case addrs.AbsResourceInstance:
				targetAddr = instance.ContainingResource().Config()
			case addrs.ModuleInstance:
				targetAddr = instance.Module()
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
