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

	for _, ms := range t.State.Modules {
		if err := t.transform(g, ms); err != nil {
			return err
		}
	}
	return nil
}

func (t *OrphanOutputTransformer) transform(g *Graph, ms *ModuleState) error {
	if ms == nil {
		return nil
	}

	path := normalizeModulePath(ms.Path)

	// Get the config for this path, which is nil if the entire module has been
	// removed.
	var c *config.Config
	if m := t.Module.Child(path[1:]); m != nil {
		c = m.Config()
	}

	// add all the orphaned outputs to the graph
	for _, n := range ms.RemovedOutputs(c) {
		g.Add(&NodeOutputOrphan{OutputName: n, PathValue: path})

	}

	return nil
}
