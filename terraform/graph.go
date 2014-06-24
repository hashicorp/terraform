package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
)

// GraphRootNode is the name of the root node in the Terraform resource
// graph. This node is just a placemarker and has no associated functionality.
const GraphRootNode = "root"

// Graph builds a dependency graph for the given configuration and state.
//
// This dependency graph shows the correct order that any resources need
// to be operated on.
//
// The Meta field of a graph Noun can contain one of the follow types. A
// description is next to each type to explain what it is.
//
//   *config.Resource - A resource itself
//   *config.ProviderConfig - The configuration for a provider that
//     should be initialized.
//   *ResourceState - An orphan resource that we only have the state of
//     and no more configuration.
//
func Graph(c *config.Config, s *State) *depgraph.Graph {
	g := new(depgraph.Graph)

	// First, build the initial resource graph. This only has the resources
	// and no dependencies.
	graphAddConfigResources(g, c)

	// Next, add the state orphans if we have any
	if s != nil {
		graphAddOrphans(g, c, s)
	}

	// Map the provider configurations to all of the resources
	graphAddProviderConfigs(g, c)

	// Add all the variable dependencies
	graphAddVariableDeps(g)

	// Build the root so that we have a single valid root
	graphAddRoot(g)

	return g
}

// configGraph turns a configuration structure into a dependency graph.
func graphAddConfigResources(g *depgraph.Graph, c *config.Config) {
	// This tracks all the resource nouns
	nouns := make(map[string]*depgraph.Noun)
	for _, r := range c.Resources {
		noun := &depgraph.Noun{
			Name: r.Id(),
			Meta: r,
		}
		nouns[noun.Name] = noun
	}

	// Build the list of nouns that we iterate over
	nounsList := make([]*depgraph.Noun, 0, len(nouns))
	for _, n := range nouns {
		nounsList = append(nounsList, n)
	}

	g.Name = "terraform"
	g.Nouns = append(g.Nouns, nounsList...)
}

// graphAddOrphans adds the orphans to the graph.
func graphAddOrphans(g *depgraph.Graph, c *config.Config, s *State) {
	for _, k := range s.Orphans(c) {
		rs := s.Resources[k]
		noun := &depgraph.Noun{
			Name: k,
			Meta: rs,
		}
		g.Nouns = append(g.Nouns, noun)
	}
}

// graphAddProviderConfigs cycles through all the resource-like nodes
// and adds the provider configuration nouns into the tree.
func graphAddProviderConfigs(g *depgraph.Graph, c *config.Config) {
	nounsList := make([]*depgraph.Noun, 0, 2)
	pcNouns := make(map[string]*depgraph.Noun)
	for _, noun := range g.Nouns {
		var rtype string
		switch m := noun.Meta.(type) {
		case *config.Resource:
			rtype = m.Type
		case *ResourceState:
			rtype = m.Type
		default:
			continue
		}

		// Look up the provider config for this resource
		pcName := config.ProviderConfigName(rtype, c.ProviderConfigs)
		if pcName == "" {
			continue
		}

		// We have one, so build the noun if it hasn't already been made
		pcNoun, ok := pcNouns[pcName]
		if !ok {
			pcNoun = &depgraph.Noun{
				Name: fmt.Sprintf("provider.%s", pcName),
				Meta: c.ProviderConfigs[pcName],
			}
			pcNouns[pcName] = pcNoun
			nounsList = append(nounsList, pcNoun)
		}

		dep := &depgraph.Dependency{
			Name:   pcName,
			Source: noun,
			Target: pcNoun,
		}
		noun.Deps = append(noun.Deps, dep)
	}

	// Add all the provider config nouns to the graph
	g.Nouns = append(g.Nouns, nounsList...)
}

// graphAddRoot adds a root element to the graph so that there is a single
// root to point to all the dependencies.
func graphAddRoot(g *depgraph.Graph) {
	root := &depgraph.Noun{Name: GraphRootNode}
	for _, n := range g.Nouns {
		root.Deps = append(root.Deps, &depgraph.Dependency{
			Name:   n.Name,
			Source: root,
			Target: n,
		})
	}
	g.Nouns = append(g.Nouns, root)
}

// graphAddVariableDeps inspects all the nouns and adds any dependencies
// based on variable values.
func graphAddVariableDeps(g *depgraph.Graph) {
	for _, n := range g.Nouns {
		var vars map[string]config.InterpolatedVariable
		switch m := n.Meta.(type) {
		case *config.Resource:
			vars = m.RawConfig.Variables
		case *config.ProviderConfig:
			vars = m.RawConfig.Variables
		default:
			continue
		}

		for _, v := range vars {
			// Only resource variables impose dependencies
			rv, ok := v.(*config.ResourceVariable)
			if !ok {
				continue
			}

			// Find the target
			var target *depgraph.Noun
			for _, n := range g.Nouns {
				if n.Name == rv.ResourceId() {
					target = n
					break
				}
			}

			// Build the dependency
			dep := &depgraph.Dependency{
				Name:   rv.ResourceId(),
				Source: n,
				Target: target,
			}

			n.Deps = append(n.Deps, dep)
		}
	}
}
