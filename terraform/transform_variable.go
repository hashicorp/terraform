package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// RootVariableTransformer is a GraphTransformer that adds all the root
// variables to the graph.
//
// Root variables are currently no-ops but they must be added to the
// graph since downstream things that depend on them must be able to
// reach them.
type RootVariableTransformer struct {
	Module *module.Tree
}

func (t *RootVariableTransformer) Transform(g *Graph) error {
	// If no config, no variables
	if t.Module == nil {
		return nil
	}

	// If we have no vars, we're done!
	vars := t.Module.Config().Variables
	if len(vars) == 0 {
		return nil
	}

	// Add all variables here
	for _, v := range vars {
		node := &NodeRootVariable{
			Config: v,
		}

		// Add it!
		g.Add(node)
	}

	return nil
}
