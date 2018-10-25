package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
)

// RemovedModuleTransformer implements GraphTransformer to add nodes indicating
// when a module was removed from the configuration.
type RemovedModuleTransformer struct {
	Config *configs.Config // root node in the config tree
	State  *states.State
}

func (t *RemovedModuleTransformer) Transform(g *Graph) error {
	// nothing to remove if there's no state!
	if t.State == nil {
		return nil
	}

	for _, m := range t.State.Modules {
		cc := t.Config.DescendentForInstance(m.Addr)
		if cc != nil {
			continue
		}

		log.Printf("[DEBUG] %s is no longer in configuration\n", m.Addr)
		g.Add(&NodeModuleRemoved{Addr: m.Addr})
	}
	return nil
}
