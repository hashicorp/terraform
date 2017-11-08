package terraform

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

type RemovedModuleTransformer struct {
	Module *module.Tree // root module
	State  *State
}

func (t *RemovedModuleTransformer) Transform(g *Graph) error {
	// nothing to remove if there's no state!
	if t.State == nil {
		return nil
	}

	// get a map of all nodes by path, so we can connect anything that might be
	// in the module
	refMap := map[string][]dag.Vertex{}
	for _, v := range g.Vertices() {
		if pn, ok := v.(GraphNodeSubPath); ok {
			path := normalizeModulePath(pn.Path())[1:]
			p := modulePrefixStr(path)
			refMap[p] = append(refMap[p], v)
		}
	}

	for _, m := range t.State.Modules {
		c := t.Module.Child(m.Path[1:])
		if c != nil {
			continue
		}

		log.Printf("[DEBUG] module %s no longer in config\n", modulePrefixStr(m.Path))

		node := &NodeModuleRemoved{PathValue: m.Path}
		g.Add(node)

		// connect this to anything that contains the module's path
		refPath := modulePrefixStr(m.Path)
		for p, nodes := range refMap {
			if strings.HasPrefix(p, refPath) {
				for _, parent := range nodes {
					g.Connect(dag.BasicEdge(node, parent))
				}
			}
		}
	}
	return nil
}
