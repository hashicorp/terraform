package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
)

// RemovedModuleTransformer implements GraphTransformer to add nodes indicating
// when a module was removed from the configuration.
type RemovedModuleTransformer struct {
	Config *configs.Config // root node in the config tree
	State  *State
}

func (t *RemovedModuleTransformer) Transform(g *Graph) error {
	// nothing to remove if there's no state!
	if t.State == nil {
		return nil
	}

	for _, m := range t.State.Modules {
		path := normalizeModulePath(m.Path)
		cc := t.Config.DescendentForInstance(path)
		if cc != nil {
			continue
		}

		log.Printf("[DEBUG] %s is no longer in configuration\n", path)
		g.Add(&NodeModuleRemoved{Addr: path})
	}
	return nil
}
