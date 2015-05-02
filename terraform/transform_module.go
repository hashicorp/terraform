package terraform

import (
	"fmt"
	"github.com/hashicorp/terraform/dag"
)

// ModuleInputTransformer is a GraphTransformer that adds a node to the
// graph for setting the module input variables for the remainder of the
// graph.
type ModuleInputTransformer struct {
	Variables map[string]string
}

func (t *ModuleInputTransformer) Transform(g *Graph) error {
	// Create the node
	n := &graphNodeModuleInput{Variables: t.Variables}

	// Add it to the graph
	g.Add(n)

	// Connect the inputs to the bottom of the graph so that it happens
	// first.
	for _, v := range g.Vertices() {
		if v == n {
			continue
		}

		if g.DownEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(v, n))
		}
	}

	return nil
}

// ModuleDestroyTransformer is a GraphTransformer that adds a node
// to the graph that will just mark the full module for destroy in
// the destroy scenario.
type ModuleDestroyTransformer struct{}

func (t *ModuleDestroyTransformer) Transform(g *Graph) error {
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

type graphNodeModuleInput struct {
	Variables map[string]string
}

func (n *graphNodeModuleInput) Name() string {
	return "module inputs"
}

// GraphNodeEvalable impl.
func (n *graphNodeModuleInput) EvalTree() EvalNode {
	return &EvalSetVariables{Variables: n.Variables}
}

// graphNodeModuleSkippable impl.
func (n *graphNodeModuleInput) FlattenSkip() bool {
	return true
}
