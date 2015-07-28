package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeNoopPrunable can be implemented by nodes that can be
// pruned if they are noops.
type GraphNodeNoopPrunable interface {
	Noop(*NoopOpts) bool
}

// NoopOpts are the options available to determine if your node is a noop.
type NoopOpts struct {
	Graph    *Graph
	Vertex   dag.Vertex
	Diff     *Diff
	State    *State
	ModDiff  *ModuleDiff
	ModState *ModuleState
}

// PruneNoopTransformer is a graph transform that prunes nodes that
// consider themselves no-ops. This is done to both simplify the graph
// as well as to remove graph nodes that might otherwise cause problems
// during the graph run. Therefore, this transformer isn't completely
// an optimization step, and can instead be considered critical to
// Terraform operations.
//
// Example of the above case: variables for modules interpolate their values.
// Interpolation will fail on destruction (since attributes are being deleted),
// but variables shouldn't even eval if there is nothing that will consume
// the variable. Therefore, variables can note that they can be omitted
// safely in this case.
//
// The PruneNoopTransformer will prune nodes depth first, and will automatically
// create connect through the dependencies of pruned nodes. For example,
// if we have a graph A => B => C (A depends on B, etc.), and B decides to
// be removed, we'll still be left with A => C; the edge will be properly
// connected.
type PruneNoopTransformer struct {
	Diff  *Diff
	State *State
}

func (t *PruneNoopTransformer) Transform(g *Graph) error {
	// Find the leaves.
	leaves := make([]dag.Vertex, 0, 10)
	for _, v := range g.Vertices() {
		if g.DownEdges(v).Len() == 0 {
			leaves = append(leaves, v)
		}
	}

	// Do a depth first walk from the leaves and remove things.
	return g.ReverseDepthFirstWalk(leaves, func(v dag.Vertex, depth int) error {
		// We need a prunable
		pn, ok := v.(GraphNodeNoopPrunable)
		if !ok {
			return nil
		}

		// Start building the noop opts
		path := g.Path
		if pn, ok := v.(GraphNodeSubPath); ok {
			path = pn.Path()
		}

		var modDiff *ModuleDiff
		var modState *ModuleState
		if t.Diff != nil {
			modDiff = t.Diff.ModuleByPath(path)
		}
		if t.State != nil {
			modState = t.State.ModuleByPath(path)
		}

		// Determine if its a noop. If it isn't, just return
		noop := pn.Noop(&NoopOpts{
			Graph:    g,
			Vertex:   v,
			Diff:     t.Diff,
			State:    t.State,
			ModDiff:  modDiff,
			ModState: modState,
		})
		if !noop {
			return nil
		}

		// It is a noop! We first preserve edges.
		up := g.UpEdges(v).List()
		for _, downV := range g.DownEdges(v).List() {
			for _, upV := range up {
				g.Connect(dag.BasicEdge(upV, downV))
			}
		}

		// Then remove it
		g.Remove(v)

		return nil
	})
}
