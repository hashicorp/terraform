package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
)

// OrphanOutputTransformer finds the outputs that aren't present
// in the given config that are in the state and adds them to the graph
// for deletion.
type OrphanOutputTransformer struct {
	Config *configs.Config // Root of config tree
	State  *states.State   // State is the root state
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

func (t *OrphanOutputTransformer) transform(g *Graph, ms *states.Module) error {
	if ms == nil {
		return nil
	}

	moduleAddr := ms.Addr

	// Get the config for this path, which is nil if the entire module has been
	// removed.
	var outputs map[string]*configs.Output
	if c := t.Config.DescendentForInstance(moduleAddr); c != nil {
		outputs = c.Module.Outputs
	}

	// An output is "orphaned" if it's present in the state but not declared
	// in the configuration.
	for name := range ms.OutputValues {
		if _, exists := outputs[name]; exists {
			continue
		}

		g.Add(&NodeOutputOrphan{
			Addr: addrs.OutputValue{Name: name}.Absolute(moduleAddr),
		})
	}

	return nil
}
