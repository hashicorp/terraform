package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
)

// OrphanOutputTransformer finds the outputs that aren't present
// in the given config that are in the state and adds them to the graph
// for deletion.
type OrphanOutputTransformer struct {
	Config *configs.Config // Root of config tree
	State  *State          // State is the root state
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

	moduleAddr := normalizeModulePath(ms.Path)

	// Get the config for this path, which is nil if the entire module has been
	// removed.
	var outputs map[string]*configs.Output
	if c := t.Config.DescendentForInstance(moduleAddr); c != nil {
		outputs = c.Module.Outputs
	}

	// add all the orphaned outputs to the graph
	for _, addr := range ms.RemovedOutputs(outputs) {
		g.Add(&NodeOutputOrphan{
			Addr: addr.Absolute(moduleAddr),
		})
	}

	return nil
}
