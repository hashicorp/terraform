package terraform

import (
	"fmt"
	"sort"
	"strings"

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
	ID           string
	Providers    map[string]ResourceProvider
	ProviderKeys []string
	Config       *config.ProviderConfig
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
	graphAddConfigResources(g, c, s)

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

// GraphFull completes the raw graph returned by Graph by initializing
// all the resource providers.
//
// This may add new nodes to the graph, since it can add new resource
// providers based on the mapping given in the case that a provider
// configuration was not specified.
//
// Various errors can be returned from this function, such as if there
// is no matching provider for a resource, a resource provider can't be
// created, etc.
func GraphFull(g *depgraph.Graph, ps map[string]ResourceProviderFactory) error {
	// Add missing providers from the mapping
	if err := graphAddMissingResourceProviders(g, ps); err != nil {
		return err
	}

	// Initialize all the providers
	if err := graphInitResourceProviders(g, ps); err != nil {
		return err
	}

	// Map the providers to resources
	if err := graphMapResourceProviders(g); err != nil {
		return err
	}

	return nil
}

// configGraph turns a configuration structure into a dependency graph.
func graphAddConfigResources(
	g *depgraph.Graph, c *config.Config, s *State) {
	// This tracks all the resource nouns
	nouns := make(map[string]*depgraph.Noun)
	for _, r := range c.Resources {
		var state *ResourceState
		if s != nil {
			state = s.Resources[r.Id()]
		}

		noun := &depgraph.Noun{
			Name: r.Id(),
			Meta: &GraphNodeResource{
				Type:   r.Type,
				Config: r,
				Resource: &Resource{
					Id:    r.Id(),
					State: state,
				},
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

// graphAddMissingResourceProviders adds GraphNodeResourceProvider nodes for
// the resources that do not have an explicit resource provider specified
// because no provider configuration was given.
func graphAddMissingResourceProviders(
	g *depgraph.Graph,
	ps map[string]ResourceProviderFactory) error {
	var errs []error

	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.ResourceProviderID != "" {
			continue
		}

		prefixes := matchingPrefixes(rn.Type, ps)
		if len(prefixes) == 0 {
			errs = append(errs, fmt.Errorf(
				"No matching provider for type: %s",
				rn.Type))
			continue
		}

		// The resource provider ID is simply the shortest matching
		// prefix, since that'll give us the most resource providers
		// to choose from.
		rn.ResourceProviderID = prefixes[len(prefixes)-1]

		// If we don't have a matching noun for this yet, insert it.
		pn := g.Noun(fmt.Sprintf("provider.%s", rn.ResourceProviderID))
		if pn == nil {
			pn = &depgraph.Noun{
				Name: fmt.Sprintf("provider.%s", rn.ResourceProviderID),
				Meta: &GraphNodeResourceProvider{
					ID:     rn.ResourceProviderID,
					Config: nil,
				},
			}
			g.Nouns = append(g.Nouns, pn)
		}

		// Add the provider configuration noun as a dependency
		dep := &depgraph.Dependency{
			Name:   pn.Name,
			Source: n,
			Target: pn,
		}
		n.Deps = append(n.Deps, dep)
	}

	if len(errs) > 0 {
		return &MultiError{Errors: errs}
	}

	return nil
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
					Id:    k,
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
			target := g.Noun(rv.ResourceId())
			if target == nil {
				continue
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

// graphInitResourceProviders maps the resource providers onto the graph
// given a mapping of prefixes to resource providers.
//
// Unlike the graphAdd* functions, this one can return an error if resource
// providers can't be found or can't be instantiated.
func graphInitResourceProviders(
	g *depgraph.Graph,
	ps map[string]ResourceProviderFactory) error {
	var errs []error

	// Keep track of providers we know we couldn't instantiate so
	// that we don't get a ton of errors about the same provider.
	failures := make(map[string]struct{})

	for _, n := range g.Nouns {
		// We only care about the resource providers first. There is guaranteed
		// to be only one node per tuple (providerId, providerConfig), which
		// means we don't need to verify we have instantiated it before.
		rn, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}

		prefixes := matchingPrefixes(rn.ID, ps)
		if len(prefixes) > 0 {
			if _, ok := failures[prefixes[0]]; ok {
				// We already failed this provider, meaning this
				// resource will never succeed, so just continue.
				continue
			}
		}

		// Go through each prefix and instantiate if necessary, then
		// verify if this provider is of use to us or not.
		rn.Providers = make(map[string]ResourceProvider)
		rn.ProviderKeys = prefixes
		for _, prefix := range prefixes {
			p, err := ps[prefix]()
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"Error instantiating resource provider for "+
						"prefix %s: %s", prefix, err))

				// Record the error so that we don't check it again
				failures[prefix] = struct{}{}

				// Jump to the next prefix
				continue
			}

			rn.Providers[prefix] = p
		}

		// If we never found a provider, then error and continue
		if len(rn.Providers) == 0 {
			errs = append(errs, fmt.Errorf(
				"Provider for configuration '%s' not found.",
				rn.ID))
			continue
		}
	}

	if len(errs) > 0 {
		return &MultiError{Errors: errs}
	}

	return nil
}

// graphMapResourceProviders takes a graph that already has initialized
// the resource providers (using graphInitResourceProviders) and maps the
// resource providers to the resources themselves.
func graphMapResourceProviders(g *depgraph.Graph) error {
	var errs []error

	// First build a mapping of resource provider ID to the node that
	// contains those resources.
	mapping := make(map[string]*GraphNodeResourceProvider)
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}
		mapping[rn.ID] = rn
	}

	// Now go through each of the resources and find a matching provider.
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}

		rpn, ok := mapping[rn.ResourceProviderID]
		if !ok {
			// This should never happen since when building the graph
			// we ensure that everything matches up.
			panic(fmt.Sprintf(
				"Resource provider ID not found: %s (type: %s)",
				rn.ResourceProviderID,
				rn.Type))
		}

		var provider ResourceProvider
		for _, k := range rpn.ProviderKeys {
			// Only try this provider if it has the right prefix
			if !strings.HasPrefix(rn.Type, k) {
				continue
			}

			rp := rpn.Providers[k]
			if ProviderSatisfies(rp, rn.Type) {
				provider = rp
				break
			}
		}

		if provider == nil {
			errs = append(errs, fmt.Errorf(
				"Resource provider not found for resource type '%s'",
				rn.Type))
			continue
		}

		rn.Resource.Provider = provider
	}

	if len(errs) > 0 {
		return &MultiError{Errors: errs}
	}

	return nil
}

// matchingPrefixes takes a resource type and a set of resource
// providers we know about by prefix and returns a list of prefixes
// that might be valid for that resource.
//
// The list returned is in the order that they should be attempted.
func matchingPrefixes(
	t string,
	ps map[string]ResourceProviderFactory) []string {
	result := make([]string, 0, 1)
	for prefix, _ := range ps {
		if strings.HasPrefix(t, prefix) {
			result = append(result, prefix)
		}
	}

	// Sort by longest first
	sort.Sort(stringLenSort(result))

	return result
}

// stringLenSort implements sort.Interface and sorts strings in increasing
// length order. i.e. "a", "aa", "aaa"
type stringLenSort []string

func (s stringLenSort) Len() int {
	return len(s)
}

func (s stringLenSort) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func (s stringLenSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
