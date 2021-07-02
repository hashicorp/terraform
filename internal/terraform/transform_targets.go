package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	SetTargets([]addrs.Targetable)
}

// TargetsTransformer is a GraphTransformer that, when the user specifies a
// list of resources to target, limits the graph to only those resources and
// their dependencies.
type TargetsTransformer struct {
	// List of targeted resource names specified by the user
	Targets []addrs.Targetable
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
				if v, ok := v.(graphNodeExemptFromTarget); ok && v.NodeExemptFromTarget() {
					continue // skip removing this one, then
				}
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
		isTargeted := t.nodeIsTarget(v, addrs)
		isExempt := false
		if v, ok := v.(graphNodeExemptFromTarget); ok {
			isExempt = v.NodeExemptFromTarget()
		}

		if isTargeted {
			targetedNodes.Add(v)

			deps, _ := g.Ancestors(v)
			for _, d := range deps {
				targetedNodes.Add(d)
			}
		}

		// If the node is either directly targeted or exempt from targets
		// then it might need to know the set of target addresses. This is
		// important for nodes that use DynamicExpand, so that they can
		// also apply the same filter to their dynamic subgraphs. It's
		// also important for "target-exempt" nodes because they must
		// handle the situation where some of their dependencies might
		// get removed from the graph.
		if isTargeted || isExempt {
			if tn, ok := v.(GraphNodeTargetable); ok {
				tn.SetTargets(addrs)
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
		// No other node types can be directly targeted, so they will
		// be selected only indirectly by a targeted node depending on them.
		return false
	}

	for _, targetAddr := range targets {
		switch vertexAddr.(type) {
		case addrs.ConfigResource:
			// Before expansion happens, we only have nodes that know their
			// ConfigResource address.  We need to take the more specific
			// target addresses and generalize them in order to compare with a
			// ConfigResource.
			switch target := targetAddr.(type) {
			case addrs.AbsResourceInstance:
				targetAddr = target.ContainingResource().Config()
			case addrs.AbsResource:
				targetAddr = target.Config()
			case addrs.ModuleInstance:
				targetAddr = target.Module()
			}
		}

		if targetAddr.TargetContains(vertexAddr) {
			return true
		}
	}

	return false
}

// graphNodeExemptFromTarget is an interface implemented by notes that might
// exclude themselves from being removed by targeting even if they are not
// identified as a target.
//
// Being exempt from targeting at all is a different idea than being selected
// by the target: any dependencies of an exempt node are still subject to
// being removed by targeting, and so the exempt node must check to see
// whether each of its dependencies was actually included in the targets,
// rather than assuming.
type graphNodeExemptFromTarget interface {
	NodeExemptFromTarget() bool

	// A node which excludes itself should typically do something to handle
	// targeting itself when executed or dynamically expanded, and so to remind
	// about that any exempt nodes must also implement GraphNodeTargetable in
	// order to obtain the targets during graph construction.
	GraphNodeTargetable
}
