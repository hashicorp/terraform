package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// OrphanOutputTransformer finds the outputs that aren't present
// in the given config that are in the state and adds them to the graph
// for deletion.
type OrphanOutputTransformer struct {
	Module *module.Tree // Root module
	State  *State       // State is the root state
}

func (t *OrphanOutputTransformer) Transform(g *Graph) error {
	if t.State == nil {
		log.Printf("[DEBUG] No state, no orphan outputs")
		return nil
	}

	return t.transform(g, t.Module)
}

func (t *OrphanOutputTransformer) transform(g *Graph, m *module.Tree) error {
	// Get our configuration, and recurse into children
	var c *config.Config
	if m != nil {
		c = m.Config()
		for _, child := range m.Children() {
			if err := t.transform(g, child); err != nil {
				return err
			}
		}
	}

	// Get the state. If there is no state, then we have no orphans!
	path := normalizeModulePath(m.Path())
	state := t.State.ModuleByPath(path)
	if state == nil {
		return nil
	}

	// Make a map of the valid outputs
	valid := make(map[string]struct{})
	for _, o := range c.Outputs {
		valid[o.Name] = struct{}{}
	}

	// Go through the outputs and find the ones that aren't in our config.
	for n, _ := range state.Outputs {
		// If it is in the valid map, then ignore
		if _, ok := valid[n]; ok {
			continue
		}

		// Orphan!
		g.Add(&NodeOutputOrphan{OutputName: n, PathValue: path})
	}

	return nil
}
