package terraform

import (
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
