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
// This only adds variables that either have no dependencies (and therefore
// always succeed) or has dependencies that are 100% represented in the
// graph.
type ModuleVariableTransformer struct {
	Module *module.Tree
}

func (t *ModuleVariableTransformer) Transform(g *Graph) error {
	return t.transform(g, nil, t.Module)
}

func (t *ModuleVariableTransformer) transform(g *Graph, parent, m *module.Tree) error {
	// If no config, no variables
	if m == nil {
		return nil
	}

	// If we have no parent, then don't do anything. This is because
	// we need to be able to get the set value from the module declaration.
	if err := t.transformSingle(g, parent, m); err != nil {
		return nil
	}

	// Transform all the children. This has to be _after_ the above
	// since children can reference parent variables but parents can't
	// access children. Example:
	//
	//   module foo { value = "${var.foo}" }
	//
	// The "value" var in "foo" (a child) is accessing the "foo" bar
	// in the parent (current module). However, there is no way for the
	// current module to reference a variable in the child module.
	for _, c := range m.Children() {
		if err := t.transform(g, m, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ModuleVariableTransformer) transformSingle(g *Graph, parent, m *module.Tree) error {
	// If we have no parent, we can't determine if the parent uses our variables
	if parent == nil {
		return nil
	}

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

		// If the node references something, then we check to make sure
		// that the thing it references is in the graph. If it isn't, then
		// we don't add it because we may not be able to compute the output.
		//
		// If the node references nothing, we always include it since there
		// is no other clear time to compute it.
		matches, missing := refMap.References(node)
		if len(missing) > 0 {
			log.Printf(
				"[INFO] Not including %q in graph, matches: %v, missing: %s",
				dag.VertexName(node), matches, missing)
			continue
		}

		// Add it!
		g.Add(node)
	}

	return nil
}
