package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

type ModuleTransformer struct {
	Config    *configs.Config
	Installer configs.ModuleWalker
}

func (t *ModuleTransformer) Transform(graph *Graph) error {
	if t.Config == nil {
		return nil
	}

	for _, call := range t.Config.Module.ModuleCalls {
		instancePath := graph.Path.Child(call.Name, addrs.NoKey)
		// instancePath := graph.Path.Module().Child(call.Name)

		err := t.transform(graph, t.Config, instancePath, call)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *ModuleTransformer) transform(graph *Graph, cfg *configs.Config, path addrs.ModuleInstance, modCall *configs.ModuleCall) error {
	n := &nodeInstallModule{
		Addr:       path,
		ModuleCall: modCall,
		Parent:     cfg,
		Installer:  t.Installer,
	}
	var installNode dag.Vertex = n
	graph.Add(installNode)
	log.Printf("[TRACE] ModuleTransformer: Added %s as %T", path, installNode)

	return nil
}
