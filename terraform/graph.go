package terraform

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
)

// GraphOpts are options used to create the resource graph that Terraform
// walks to make changes to infrastructure.
//
// Depending on what options are set, the resulting graph will come in
// varying degrees of completeness.
type GraphOpts struct {
	// Config is the configuration from which to build the basic graph.
	// This is the only required item.
	Config *config.Config

	// Diff of changes that will be applied to the given state. This will
	// associate a ResourceDiff with applicable resources. Additionally,
	// new resource nodes representing resource destruction may be inserted
	// into the graph.
	Diff *Diff

	// State, if present, will make the ResourceState available on each
	// resource node. Additionally, any orphans will be added automatically
	// to the graph.
	State *State

	// Providers is a mapping of prefixes to a resource provider. If given,
	// resource providers will be found, initialized, and associated to the
	// resources in the graph.
	//
	// This will also potentially insert new nodes into the graph for
	// the configuration of resource providers.
	Providers map[string]ResourceProviderFactory
}

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

// Graph builds a dependency graph of all the resources for infrastructure
// change.
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
func Graph(opts *GraphOpts) (*depgraph.Graph, error) {
	if opts.Config == nil {
		return nil, errors.New("Config is required for Graph")
	}

	g := new(depgraph.Graph)

	// First, build the initial resource graph. This only has the resources
	// and no dependencies.
	graphAddConfigResources(g, opts.Config, opts.State)

	// Next, add the state orphans if we have any
	if opts.State != nil {
		graphAddOrphans(g, opts.Config, opts.State)
	}

	// Map the provider configurations to all of the resources
	graphAddProviderConfigs(g, opts.Config)

	// Add all the variable dependencies
	graphAddVariableDeps(g)

	// Build the root so that we have a single valid root
	graphAddRoot(g)

	// If providers were given, lets associate the proper providers and
	// instantiate them.
	if len(opts.Providers) > 0 {
		// Add missing providers from the mapping
		if err := graphAddMissingResourceProviders(g, opts.Providers); err != nil {
			return nil, err
		}

		// Initialize all the providers
		if err := graphInitResourceProviders(g, opts.Providers); err != nil {
			return nil, err
		}

		// Map the providers to resources
		if err := graphMapResourceProviders(g); err != nil {
			return nil, err
		}
	}

	// If we have a diff, then make sure to add that in
	if opts.Diff != nil {
		if err := graphAddDiff(g, opts.Diff); err != nil {
			return nil, err
		}
	}

	// Validate
	if err := g.Validate(); err != nil {
		return nil, err
	}

	return g, nil
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
		if state == nil {
			state = &ResourceState{
				Type: r.Type,
			}
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

// graphAddDiff takes an already-built graph of resources and adds the
// diffs to the resource nodes themselves.
//
// This may also introduces new graph elements. If there are diffs that
// require a destroy, new elements may be introduced since destroy order
// is different than create order. For example, destroying a VPC requires
// destroying the VPC's subnets first, whereas creating a VPC requires
// doing it before the subnets are created. This function handles inserting
// these nodes for you.
func graphAddDiff(g *depgraph.Graph, d *Diff) error {
	var nlist []*depgraph.Noun
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}

		rd, ok := d.Resources[rn.Resource.Id]
		if !ok {
			continue
		}
		if rd.Empty() {
			continue
		}

		if rd.Destroy || rd.RequiresNew() {
			// If we're destroying, we create a new destroy node with
			// the proper dependencies. Perform a dirty copy operation.
			newNode := new(GraphNodeResource)
			*newNode = *rn
			newNode.Resource = new(Resource)
			*newNode.Resource = *rn.Resource

			// Make the diff _just_ the destroy.
			newNode.Resource.Diff = &ResourceDiff{Destroy: true}

			// Append it to the list so we handle it later
			deps := make([]*depgraph.Dependency, len(n.Deps))
			copy(deps, n.Deps)
			newN := &depgraph.Noun{
				Name: fmt.Sprintf("%s (destroy)", newNode.Resource.Id),
				Meta: newNode,
				Deps: deps,
			}
			nlist = append(nlist, newN)

			// Mark the old diff to not destroy since we handle that in
			// the dedicated node.
			rd.Destroy = false

			// Add to the new noun to our dependencies so that the destroy
			// happens before the apply.
			n.Deps = append(n.Deps, &depgraph.Dependency{
				Name:   newN.Name,
				Source: n,
				Target: newN,
			})
		}

		rn.Resource.Diff = rd
	}

	// Go through each noun and make sure we calculate all the dependencies
	// properly.
	for _, n := range nlist {
		rn := n.Meta.(*GraphNodeResource)

		// If we have no dependencies, then just continue
		deps := rn.Resource.State.Dependencies
		if len(deps) == 0 {
			continue
		}

		// We have dependencies. We must be destroyed BEFORE those
		// dependencies. Look to see if they're managed.
		for _, dep := range deps {
			for _, n2 := range nlist {
				rn2 := n2.Meta.(*GraphNodeResource)
				if rn2.Resource.State.ID == dep.ID {
					n2.Deps = append(n2.Deps, &depgraph.Dependency{
						Name:   n.Name,
						Source: n2,
						Target: n,
					})

					break
				}
			}
		}
	}

	// Add the nouns to the graph
	g.Nouns = append(g.Nouns, nlist...)

	return nil
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
