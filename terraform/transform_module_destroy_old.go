package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// ModuleDestroyTransformer is a GraphTransformer that adds a node
// to the graph that will just mark the full module for destroy in
// the destroy scenario.
type ModuleDestroyTransformerOld struct{}

func (t *ModuleDestroyTransformerOld) Transform(g *Graph) error {
	// Create the node
	n := &graphNodeModuleDestroy{Path: g.Path}

	// Add it to the graph. We don't need any edges because
	// it can happen whenever.
	g.Add(n)

	return nil
}

type graphNodeModuleDestroy struct {
	Path []string
}

func (n *graphNodeModuleDestroy) Name() string {
	return "plan-destroy"
}

// GraphNodeEvalable impl.
func (n *graphNodeModuleDestroy) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops:  []walkOperation{walkPlanDestroy},
		Node: &EvalDiffDestroyModule{Path: n.Path},
	}
}

// GraphNodeFlattenable impl.
func (n *graphNodeModuleDestroy) Flatten(p []string) (dag.Vertex, error) {
	return &graphNodeModuleDestroyFlat{
		graphNodeModuleDestroy: n,
		PathValue:              p,
	}, nil
}

type graphNodeModuleDestroyFlat struct {
	*graphNodeModuleDestroy

	PathValue []string
}

func (n *graphNodeModuleDestroyFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.graphNodeModuleDestroy.Name())
}

func (n *graphNodeModuleDestroyFlat) Path() []string {
	return n.PathValue
}
