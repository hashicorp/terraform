package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
)

// GraphRootNode is the name of the root node in the Terraform resource
// graph. This node is just a placemarker and has no associated functionality.
const GraphRootNode = "root"

// GraphNodeResource is a node type in the graph that represents a resource.
type GraphNodeResource struct {
	Type               string
	Config             *config.Resource
	Orphan             bool
	Resource           *Resource
	ResourceProviderID string
}

// GraphNodeResourceProvider is a node type in the graph that represents
// the configuration for a resource provider.
type GraphNodeResourceProvider struct {
	ID       string
	Provider ResourceProvider
	Config   *config.ProviderConfig
}

// Graph builds a dependency graph for the given configuration and state.
//
// Before using this graph, Validate should be called on it. This will perform
// some initialization necessary such as setting up a root node. This function
// doesn't perform the Validate automatically in case the caller wants to
// modify the graph.
//
// This dependency graph shows the correct order that any resources need
// to be operated on.
//
// The Meta field of a graph Noun can contain one of the follow types. A
// description is next to each type to explain what it is.
//
//   *GraphNodeResource - A resource. See the documentation of this
//     struct for more details.
//   *GraphNodeResourceProvider - A resource provider that needs to be
//     configured at this point.
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
			Meta: &GraphNodeResource{
				Type:   r.Type,
				Config: r,
			},
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
			Meta: &GraphNodeResource{
				Type:   rs.Type,
				Orphan: true,
				Resource: &Resource{
					State: rs,
				},
			},
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
		resourceNode := noun.Meta.(*GraphNodeResource)

		// Look up the provider config for this resource
		pcName := config.ProviderConfigName(resourceNode.Type, c.ProviderConfigs)
		if pcName == "" {
			continue
		}

		// We have one, so build the noun if it hasn't already been made
		pcNoun, ok := pcNouns[pcName]
		if !ok {
			pcNoun = &depgraph.Noun{
				Name: fmt.Sprintf("provider.%s", pcName),
				Meta: &GraphNodeResourceProvider{
					ID:     pcName,
					Config: c.ProviderConfigs[pcName],
				},
			}
			pcNouns[pcName] = pcNoun
			nounsList = append(nounsList, pcNoun)
		}

		// Set the resource provider ID for this noun so we can look it
		// up later easily.
		resourceNode.ResourceProviderID = pcName

		// Add the provider configuration noun as a dependency
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
		case *GraphNodeResource:
			if !m.Orphan {
				vars = m.Config.RawConfig.Variables
			}
		case *GraphNodeResourceProvider:
			vars = m.Config.RawConfig.Variables
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
