package module

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// validateProviderAlias validates that all provider alias references are
// defined at some point in the parent tree. This improves UX by catching
// alias typos at the slight cost of requiring a declaration of usage. This
// is usually a good tradeoff since not many aliases are used.
func (t *Tree) validateProviderAlias() error {
	// If we're not the root, don't perform this validation. We must be the
	// root since we require full tree visibilty.
	if len(t.path) != 0 {
		return nil
	}

	// We'll use a graph to keep track of defined aliases at each level.
	// As long as a parent defines an alias, it is okay.
	var g dag.AcyclicGraph
	t.buildProviderAliasGraph(&g, nil)

	// Go through the graph and check that the usage is all good.
	var err error
	for _, v := range g.Vertices() {
		pv, ok := v.(*providerAliasVertex)
		if !ok {
			// This shouldn't happen, just ignore it.
			continue
		}

		// If we're not using any aliases, fast track and just continue
		if len(pv.Used) == 0 {
			continue
		}

		// Grab the ancestors since we're going to have to check if our
		// parents define any of our aliases.
		var parents []*providerAliasVertex
		ancestors, _ := g.Ancestors(v)
		for _, raw := range ancestors.List() {
			if pv, ok := raw.(*providerAliasVertex); ok {
				parents = append(parents, pv)
			}
		}
		for k, _ := range pv.Used {
			// Check if we define this
			if _, ok := pv.Defined[k]; ok {
				continue
			}

			// Check for a parent
			found := false
			for _, parent := range parents {
				_, found = parent.Defined[k]
				if found {
					break
				}
			}
			if found {
				continue
			}

			// We didn't find the alias, error!
			err = multierror.Append(err, fmt.Errorf(
				"module %s: provider alias must be defined by the module: %s",
				strings.Join(pv.Path, "."), k))
		}
	}

	return err
}

func (t *Tree) buildProviderAliasGraph(g *dag.AcyclicGraph, parent dag.Vertex) {
	// Add all our defined aliases
	defined := make(map[string]struct{})
	for _, p := range t.config.ProviderConfigs {
		defined[p.FullName()] = struct{}{}
	}

	// Add all our used aliases
	used := make(map[string]struct{})
	for _, r := range t.config.Resources {
		if r.Provider != "" {
			used[r.Provider] = struct{}{}
		}
	}

	// Add it to the graph
	vertex := &providerAliasVertex{
		Path:    t.Path(),
		Defined: defined,
		Used:    used,
	}
	g.Add(vertex)

	// Connect to our parent if we have one
	if parent != nil {
		g.Connect(dag.BasicEdge(vertex, parent))
	}

	// Build all our children
	for _, c := range t.Children() {
		c.buildProviderAliasGraph(g, vertex)
	}
}

// providerAliasVertex is the vertex for the graph that keeps track of
// defined provider aliases.
type providerAliasVertex struct {
	Path    []string
	Defined map[string]struct{}
	Used    map[string]struct{}
}
