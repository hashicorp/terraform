package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// OutputTransformer is a GraphTransformer that adds all the outputs
// in the configuration to the graph.
//
// This is done for the apply graph builder even if dependent nodes
// aren't changing since there is no downside: the state will be available
// even if the dependent items aren't changing.
type OutputTransformer struct {
	Module *module.Tree
}

func (t *OutputTransformer) Transform(g *Graph) error {
	return nil
}
