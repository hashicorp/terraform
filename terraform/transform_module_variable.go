package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// ModuleVariableTransformer is a GraphTransformer that adds all the variables
// in the configuration to the graph.
//
// This only adds variables that are referenced by other things in the graph.
// If a module variable is not referenced, it won't be added to the graph.
type ModuleVariableTransformer struct {
	Module *module.Tree

	DisablePrune bool // True if pruning unreferenced should be disabled
}

func (t *ModuleVariableTransformer) Transform(g *Graph) error {
	return t.transform(g, nil, t.Module)
}

func (t *ModuleVariableTransformer) transform(g *Graph, parent, m *module.Tree) error {
	// If no config, no variables
	if m == nil {
		return nil
	}

	// Transform all the children. This must be done BEFORE the transform
	// above since child module variables can reference parent module variables.
	for _, c := range m.Children() {
		if err := t.transform(g, m, c); err != nil {
			return err
		}
	}

	// If we have a parent, we can determine if a module variable is being
	// used, so we transform this.
	if parent != nil {
		if err := t.transformSingle(g, parent, m); err != nil {
			return err
		}
	}

	return nil
}

func (t *ModuleVariableTransformer) transformSingle(g *Graph, parent, m *module.Tree) error {
	// If we have no vars, we're done!
	vars := m.Config().Variables
	if len(vars) == 0 {
		log.Printf("[TRACE] Module %#v has no variables, skipping.", m.Path())
		return nil
	}

	// Look for usage of this module
	var mod *config.Module
	for _, modUse := range parent.Config().Modules {
		if modUse.Name == m.Name() {
			mod = modUse
			break
		}
	}
	if mod == nil {
		log.Printf("[INFO] Module %#v not used, not adding variables", m.Path())
		return nil
	}

	// Build the reference map so we can determine if we're referencing things.
	refMap := NewReferenceMap(g.Vertices())

	// Add all variables here
	for _, v := range vars {
		// Determine the value of the variable. If it isn't in the
		// configuration then it was never set and that's not a problem.
		var value *config.RawConfig
		if raw, ok := mod.RawConfig.Raw[v.Name]; ok {
			var err error
			value, err = config.NewRawConfig(map[string]interface{}{
				v.Name: raw,
			})
			if err != nil {
				// This shouldn't happen because it is already in
				// a RawConfig above meaning it worked once before.
				panic(err)
			}
		}

		// Build the node.
		//
		// NOTE: For now this is just an "applyable" variable. As we build
		// new graph builders for the other operations I suspect we'll
		// find a way to parameterize this, require new transforms, etc.
		node := &NodeApplyableModuleVariable{
			PathValue: normalizeModulePath(m.Path()),
			Config:    v,
			Value:     value,
			Module:    t.Module,
		}

		if !t.DisablePrune {
			// If the node is not referenced by anything, then we don't need
			// to include it since it won't be used.
			if matches := refMap.ReferencedBy(node); len(matches) == 0 {
				log.Printf(
					"[INFO] Not including %q in graph, nothing depends on it",
					dag.VertexName(node))
				continue
			}
		}

		// Add it!
		g.Add(node)
	}

	return nil
}
