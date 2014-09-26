package terraform

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/depgraph"
	"github.com/hashicorp/terraform/helper/multierror"
)

// GraphOpts are options used to create the resource graph that Terraform
// walks to make changes to infrastructure.
//
// Depending on what options are set, the resulting graph will come in
// varying degrees of completeness.
type GraphOpts struct {
	// Config is the configuration from which to build the basic graph.
	// This is the only required item.
	//Config *config.Config

	// Module is the relative root of a module tree for this graph. This
	// is the only required item. This should always be the absolute root
	// of the tree. ModulePath below should be used to constrain the depth.
	//
	// ModulePath specifies the place in the tree where Module exists.
	// This is used for State lookups.
	Module     *module.Tree
	ModulePath []string

	// Diff of changes that will be applied to the given state. This will
	// associate a ResourceDiff with applicable resources. Additionally,
	// new resource nodes representing resource destruction may be inserted
	// into the graph.
	Diff *Diff

	// State, if present, will make the ResourceState available on each
	// resource node. Additionally, any orphans will be added automatically
	// to the graph.
	//
	// Note: the state will be modified so it is initialized with basic
	// empty states for all modules/resources in this graph. If you call prune
	// later, these will be removed, but the graph adds important metadata.
	State *State

	// Providers is a mapping of prefixes to a resource provider. If given,
	// resource providers will be found, initialized, and associated to the
	// resources in the graph.
	//
	// This will also potentially insert new nodes into the graph for
	// the configuration of resource providers.
	Providers map[string]ResourceProviderFactory

	// Provisioners is a mapping of names to a resource provisioner.
	// These must be provided to support resource provisioners.
	Provisioners map[string]ResourceProvisionerFactory

	// parent specifies the parent graph if there is one. This should not be
	// set manually.
	parent *depgraph.Graph
}

// GraphRootNode is the name of the root node in the Terraform resource
// graph. This node is just a placemarker and has no associated functionality.
const GraphRootNode = "root"

// GraphMeta is the metadata attached to the graph itself.
type GraphMeta struct {
	// ModulePath is the path of the module that this graph represents.
	ModulePath []string
}

// GraphNodeModule is a node type in the graph that represents a module
// that will be created/managed.
type GraphNodeModule struct {
	Config *config.Module
	Path   []string
	Graph  *depgraph.Graph
}

// GraphNodeResource is a node type in the graph that represents a resource
// that will be created or managed. Unlike the GraphNodeResourceMeta node,
// this represents a _single_, _resource_ to be managed, not a set of resources
// or a component of a resource.
type GraphNodeResource struct {
	Index                int
	Config               *config.Resource
	Resource             *Resource
	ResourceProviderNode string
}

// GraphNodeResourceMeta is a node type in the graph that represents the
// metadata for a resource. There will be one meta node for every resource
// in the configuration.
type GraphNodeResourceMeta struct {
	ID    string
	Name  string
	Type  string
	Count int
}

// GraphNodeResourceProvider is a node type in the graph that represents
// the configuration for a resource provider.
type GraphNodeResourceProvider struct {
	ID       string
	Provider *graphSharedProvider
}

// graphSharedProvider is a structure that stores a configuration
// with initialized providers and might be shared across different
// graphs in order to have only one instance of a provider.
type graphSharedProvider struct {
	Config       *config.ProviderConfig
	Providers    map[string]ResourceProvider
	ProviderKeys []string
	Parent       *graphSharedProvider

	parentNoun *depgraph.Noun
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
	if opts.Module == nil {
		return nil, errors.New("Module is required for Graph")
	}
	if opts.ModulePath == nil {
		opts.ModulePath = rootModulePath
	}
	if !opts.Module.Loaded() {
		return nil, errors.New("Module must be loaded")
	}

	// Get the correct module in the tree that we're looking for.
	currentModule := opts.Module
	for _, n := range opts.ModulePath[1:] {
		children := currentModule.Children()
		currentModule = children[n]
	}

	var conf *config.Config
	if currentModule != nil {
		conf = currentModule.Config()
	} else {
		conf = new(config.Config)
	}

	// Get the state and diff of the module that we're working with.
	var modDiff *ModuleDiff
	var modState *ModuleState
	if opts.Diff != nil {
		modDiff = opts.Diff.ModuleByPath(opts.ModulePath)
	}
	if opts.State != nil {
		modState = opts.State.ModuleByPath(opts.ModulePath)
	}

	log.Printf("[DEBUG] Creating graph for path: %v", opts.ModulePath)

	g := new(depgraph.Graph)
	g.Meta = &GraphMeta{
		ModulePath: opts.ModulePath,
	}

	// First, build the initial resource graph. This only has the resources
	// and no dependencies. This only adds resources that are in the config
	// and not "orphans" (that are in the state, but not in the config).
	graphAddConfigResources(g, conf, modState)

	if modState != nil {
		// Next, add the state orphans if we have any
		graphAddOrphans(g, conf, modState)

		// Add tainted resources if we have any.
		graphAddTainted(g, modState)
	}

	// Create the resource provider nodes for explicitly configured
	// providers within our graph.
	graphAddConfigProviderConfigs(g, conf)

	if opts.parent != nil {
		// Add/merge the provider configurations from the parent so that
		// we properly "inherit" providers.
		graphAddParentProviderConfigs(g, opts.parent)
	}

	// First pass matching resources to providers. This will allow us to
	// determine what providers are missing.
	graphMapResourceProviderId(g)

	if len(opts.Providers) > 0 {
		// Add missing providers from the mapping.
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

	// Add the modules that are in the configuration.
	if err := graphAddConfigModules(g, conf, opts); err != nil {
		return nil, err
	}

	if opts.State != nil {
		// Add module orphans if we have any of those
		if ms := opts.State.Children(opts.ModulePath); len(ms) > 0 {
			if err := graphAddModuleOrphans(g, conf, ms, opts); err != nil {
				return nil, err
			}
		}
	}

	// Add the provider dependencies
	graphAddResourceProviderDeps(g)

	// Now, prune the providers that we aren't using.
	graphPruneResourceProviders(g)

	// Add explicit dependsOn dependencies to the graph
	graphAddExplicitDeps(g)

	// Setup the provisioners. These may have variable dependencies,
	// and must be done before dependency setup
	if err := graphMapResourceProvisioners(g, opts.Provisioners); err != nil {
		return nil, err
	}

	// Add all the variable dependencies
	graphAddVariableDeps(g)

	// Build the root so that we have a single valid root
	graphAddRoot(g)

	// If we have a diff, then make sure to add that in
	if modDiff != nil {
		if err := graphAddDiff(g, modDiff); err != nil {
			return nil, err
		}
	}

	// Encode the dependencies
	graphEncodeDependencies(g)

	// Validate
	if err := g.Validate(); err != nil {
		return nil, err
	}

	log.Printf(
		"[DEBUG] Graph %v created and valid. %d nouns.",
		opts.ModulePath,
		len(g.Nouns))

	return g, nil
}

// graphEncodeDependencies is used to initialize a State with a ResourceState
// for every resource.
//
// This method is very important to call because it will properly setup
// the ResourceState dependency information with data from the graph. This
// allows orphaned resources to be destroyed in the proper order.
func graphEncodeDependencies(g *depgraph.Graph) {
	for _, n := range g.Nouns {
		// Ignore any non-resource nodes
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		r := rn.Resource

		// Update the dependencies
		var inject []string
		for _, dep := range n.Deps {
			switch target := dep.Target.Meta.(type) {
			case *GraphNodeResource:
				if target.Resource.Id == r.Id {
					continue
				}
				inject = append(inject, target.Resource.Id)

			case *GraphNodeResourceMeta:
				// Inject each sub-resource as a depedency
				for i := 0; i < target.Count; i++ {
					id := fmt.Sprintf("%s.%d", target.ID, i)
					inject = append(inject, id)
				}
			}
		}

		// Update the dependencies
		r.Dependencies = inject
	}
}

// graphAddConfigModules adds the modules from a configuration structure
// into the graph, expanding each to their own sub-graph.
func graphAddConfigModules(
	g *depgraph.Graph,
	c *config.Config,
	opts *GraphOpts) error {
	// Just short-circuit the whole thing if we don't have modules
	if len(c.Modules) == 0 {
		return nil
	}

	// Build the list of nouns to add to the graph
	nounsList := make([]*depgraph.Noun, 0, len(c.Modules))
	for _, m := range c.Modules {
		if n, err := graphModuleNoun(m.Name, m, g, opts); err != nil {
			return err
		} else {
			nounsList = append(nounsList, n)
		}
	}

	g.Nouns = append(g.Nouns, nounsList...)
	return nil
}

// configGraph turns a configuration structure into a dependency graph.
func graphAddConfigResources(
	g *depgraph.Graph, c *config.Config, mod *ModuleState) {
	meta := g.Meta.(*GraphMeta)

	// This tracks all the resource nouns
	nouns := make(map[string]*depgraph.Noun)
	for _, r := range c.Resources {
		resourceNouns := make([]*depgraph.Noun, r.Count)
		for i := 0; i < r.Count; i++ {
			name := r.Id()
			index := -1

			// If we have a count that is more than one, then make sure
			// we suffix with the number of the resource that this is.
			if r.Count > 1 {
				name = fmt.Sprintf("%s.%d", name, i)
				index = i
			}

			var state *ResourceState
			if mod != nil {
				// Lookup the resource state
				state = mod.Resources[name]
				if state == nil {
					if r.Count == 1 {
						// If the count is one, check the state for ".0"
						// appended, which might exist if we go from
						// count > 1 to count == 1.
						state = mod.Resources[r.Id()+".0"]
					} else if i == 0 {
						// If count is greater than one, check for state
						// with just the ID, which might exist if we go
						// from count == 1 to count > 1
						state = mod.Resources[r.Id()]
					}

					// TODO(mitchellh): If one of the above works, delete
					// the old style and just copy it to the new style.
				}
			}

			if state == nil {
				state = &ResourceState{
					Type: r.Type,
				}
			}

			flags := FlagPrimary
			if len(state.Tainted) > 0 {
				flags |= FlagHasTainted
			}

			resourceNouns[i] = &depgraph.Noun{
				Name: name,
				Meta: &GraphNodeResource{
					Index:  index,
					Config: r,
					Resource: &Resource{
						Id: name,
						Info: &InstanceInfo{
							Id:         name,
							ModulePath: meta.ModulePath,
							Type:       r.Type,
						},
						State:  state.Primary,
						Config: NewResourceConfig(r.RawConfig),
						Flags:  flags,
					},
				},
			}
		}

		// If we have more than one, then create a meta node to track
		// the resources.
		if r.Count > 1 {
			metaNoun := &depgraph.Noun{
				Name: r.Id(),
				Meta: &GraphNodeResourceMeta{
					ID:    r.Id(),
					Name:  r.Name,
					Type:  r.Type,
					Count: r.Count,
				},
			}

			// Create the dependencies on this noun
			for _, n := range resourceNouns {
				metaNoun.Deps = append(metaNoun.Deps, &depgraph.Dependency{
					Name:   n.Name,
					Source: metaNoun,
					Target: n,
				})
			}

			// Assign it to the map so that we have it
			nouns[metaNoun.Name] = metaNoun
		}

		for _, n := range resourceNouns {
			nouns[n.Name] = n
		}
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
func graphAddDiff(g *depgraph.Graph, d *ModuleDiff) error {
	var nlist []*depgraph.Noun
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.Resource.Flags&FlagTainted != 0 {
			continue
		}

		rd, ok := d.Resources[rn.Resource.Id]
		if !ok {
			continue
		}
		if rd.Empty() {
			continue
		}

		if rd.Destroy {
			// If we're destroying, we create a new destroy node with
			// the proper dependencies. Perform a dirty copy operation.
			newNode := new(GraphNodeResource)
			*newNode = *rn
			newNode.Resource = new(Resource)
			*newNode.Resource = *rn.Resource

			// Make the diff _just_ the destroy.
			newNode.Resource.Diff = &InstanceDiff{Destroy: true}

			// Create the new node
			newN := &depgraph.Noun{
				Name: fmt.Sprintf("%s (destroy)", newNode.Resource.Id),
				Meta: newNode,
			}
			newN.Deps = make([]*depgraph.Dependency, len(n.Deps))

			// Copy all the dependencies and do a fixup later
			copy(newN.Deps, n.Deps)

			// Append it to the list so we handle it later
			nlist = append(nlist, newN)

			// Mark the old diff to not destroy since we handle that in
			// the dedicated node.
			newDiff := new(InstanceDiff)
			*newDiff = *rd
			newDiff.Destroy = false
			rd = newDiff

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
	injected := make(map[*depgraph.Dependency]struct{})
	for _, n := range nlist {
		deps := n.Deps
		num := len(deps)
		for i := 0; i < num; i++ {
			dep := deps[i]

			// Check if this dependency was just injected, otherwise
			// we will incorrectly flip the depedency twice.
			if _, ok := injected[dep]; ok {
				continue
			}

			switch target := dep.Target.Meta.(type) {
			case *GraphNodeResource:
				// If the other node is also being deleted,
				// we must be deleted first. E.g. if A -> B,
				// then when we create, B is created first then A.
				// On teardown, A is destroyed first, then B.
				// Thus we must flip our depedency and instead inject
				// it on B.
				for _, n2 := range nlist {
					rn2 := n2.Meta.(*GraphNodeResource)
					if target.Resource.Id == rn2.Resource.Id {
						newDep := &depgraph.Dependency{
							Name:   n.Name,
							Source: n2,
							Target: n,
						}
						injected[newDep] = struct{}{}
						n2.Deps = append(n2.Deps, newDep)
						break
					}
				}

				// Drop the dependency. We may have created
				// an inverse depedency if the dependent resource
				// is also being deleted, but this dependence is
				// no longer required.
				deps[i], deps[num-1] = deps[num-1], nil
				num--
				i--

			case *GraphNodeResourceMeta:
				// Check if any of the resources part of the meta node
				// are being destroyed, because we must be destroyed first.
				for i := 0; i < target.Count; i++ {
					id := fmt.Sprintf("%s.%d", target.ID, i)
					for _, n2 := range nlist {
						rn2 := n2.Meta.(*GraphNodeResource)
						if id == rn2.Resource.Id {
							newDep := &depgraph.Dependency{
								Name:   n.Name,
								Source: n2,
								Target: n,
							}
							injected[newDep] = struct{}{}
							n2.Deps = append(n2.Deps, newDep)
							break
						}
					}
				}

				// Drop the dependency, since there is
				// nothing that needs to be done for a meta
				// resource on destroy.
				deps[i], deps[num-1] = deps[num-1], nil
				num--
				i--

			case *GraphNodeModule:
				// We invert any module dependencies so we're destroyed
				// first, before any modules are applied.
				newDep := &depgraph.Dependency{
					Name:   n.Name,
					Source: dep.Target,
					Target: n,
				}
				dep.Target.Deps = append(dep.Target.Deps, newDep)

				// Drop the dependency. We may have created
				// an inverse depedency if the dependent resource
				// is also being deleted, but this dependence is
				// no longer required.
				deps[i], deps[num-1] = deps[num-1], nil
				num--
				i--
			case *GraphNodeResourceProvider:
				// Keep these around, but fix up the source to be ourselves
				// rather than the old node.
				newDep := *dep
				newDep.Source = n
				deps[i] = &newDep
			default:
				panic(fmt.Errorf("Unhandled depedency type: %#v", dep.Target.Meta))
			}
		}
		n.Deps = deps[:num]
	}

	// Add the nouns to the graph
	g.Nouns = append(g.Nouns, nlist...)

	return nil
}

// graphAddExplicitDeps adds the dependencies to the graph for the explicit
// dependsOn configurations.
func graphAddExplicitDeps(g *depgraph.Graph) {
	depends := false

	rs := make(map[string]*depgraph.Noun)
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}

		rs[rn.Resource.Id] = n
		if rn.Config != nil && len(rn.Config.DependsOn) > 0 {
			depends = true
		}
	}

	// If we didn't have any dependsOn, just return
	if !depends {
		return
	}

	for _, n1 := range rs {
		rn1 := n1.Meta.(*GraphNodeResource)
		for _, d := range rn1.Config.DependsOn {
			for _, n2 := range rs {
				rn2 := n2.Meta.(*GraphNodeResource)
				if rn2.Config.Id() != d {
					continue
				}

				n1.Deps = append(n1.Deps, &depgraph.Dependency{
					Name:   d,
					Source: n1,
					Target: n2,
				})
			}
		}
	}
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
		if rn.ResourceProviderNode != "" {
			continue
		}

		prefixes := matchingPrefixes(rn.Resource.Info.Type, ps)
		if len(prefixes) == 0 {
			errs = append(errs, fmt.Errorf(
				"No matching provider for type: %s",
				rn.Resource.Info.Type))
			continue
		}

		// The resource provider ID is simply the shortest matching
		// prefix, since that'll give us the most resource providers
		// to choose from.
		id := prefixes[len(prefixes)-1]
		rn.ResourceProviderNode = fmt.Sprintf("provider.%s", id)

		// If we don't have a matching noun for this yet, insert it.
		if g.Noun(rn.ResourceProviderNode) == nil {
			pn := &depgraph.Noun{
				Name: rn.ResourceProviderNode,
				Meta: &GraphNodeResourceProvider{
					ID:       id,
					Provider: new(graphSharedProvider),
				},
			}
			g.Nouns = append(g.Nouns, pn)
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func graphAddModuleOrphans(
	g *depgraph.Graph,
	config *config.Config,
	ms []*ModuleState,
	opts *GraphOpts) error {
	// Build a lookup map for the modules we do have defined
	childrenKeys := make(map[string]struct{})
	for _, m := range config.Modules {
		childrenKeys[m.Name] = struct{}{}
	}

	// Go through each of the child modules. If we don't have it in our
	// config, it is an orphan.
	var nounsList []*depgraph.Noun
	for _, m := range ms {
		k := m.Path[len(m.Path)-1]
		if _, ok := childrenKeys[k]; ok {
			// We have this module configured
			continue
		}

		if n, err := graphModuleNoun(k, nil, g, opts); err != nil {
			return err
		} else {
			nounsList = append(nounsList, n)
		}
	}

	g.Nouns = append(g.Nouns, nounsList...)
	return nil
}

// graphAddOrphans adds the orphans to the graph.
func graphAddOrphans(g *depgraph.Graph, c *config.Config, mod *ModuleState) {
	meta := g.Meta.(*GraphMeta)

	var nlist []*depgraph.Noun
	for _, k := range mod.Orphans(c) {
		rs := mod.Resources[k]
		noun := &depgraph.Noun{
			Name: k,
			Meta: &GraphNodeResource{
				Index: -1,
				Resource: &Resource{
					Id: k,
					Info: &InstanceInfo{
						Id:         k,
						ModulePath: meta.ModulePath,
						Type:       rs.Type,
					},
					State:  rs.Primary,
					Config: NewResourceConfig(nil),
					Flags:  FlagOrphan,
				},
			},
		}

		// Append it to the list so we handle it later
		nlist = append(nlist, noun)
	}

	// Add the nouns to the graph
	g.Nouns = append(g.Nouns, nlist...)

	// Handle the orphan dependencies after adding them
	// to the graph because there may be depedencies between the
	// orphans that otherwise cannot be handled
	for _, n := range nlist {
		rn := n.Meta.(*GraphNodeResource)

		// If we have no dependencies, then just continue
		rs := mod.Resources[n.Name]
		if len(rs.Dependencies) == 0 {
			continue
		}

		for _, n2 := range g.Nouns {
			rn2, ok := n2.Meta.(*GraphNodeResource)
			if !ok {
				continue
			}

			// Don't ever depend on ourselves
			if rn2 == rn {
				continue
			}

			for _, depName := range rs.Dependencies {
				if rn2.Resource.Id != depName {
					continue
				}
				dep := &depgraph.Dependency{
					Name:   depName,
					Source: n,
					Target: n2,
				}
				n.Deps = append(n.Deps, dep)
			}
		}
	}
}

// graphAddParentProviderConfigs goes through and adds/merges provider
// configurations from the parent.
func graphAddParentProviderConfigs(g, parent *depgraph.Graph) {
	var nounsList []*depgraph.Noun
	for _, n := range parent.Nouns {
		pn, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}

		// If we have a provider configuration with the exact same
		// name, then set specify the parent pointer to their shared
		// config.
		ourProviderRaw := g.Noun(n.Name)

		// If we don't have a matching configuration, then create one.
		if ourProviderRaw == nil {
			noun := &depgraph.Noun{
				Name: n.Name,
				Meta: &GraphNodeResourceProvider{
					ID: pn.ID,
					Provider: &graphSharedProvider{
						Parent:     pn.Provider,
						parentNoun: n,
					},
				},
			}

			nounsList = append(nounsList, noun)
			continue
		}

		// If we have a matching configuration, then set the parent pointer
		ourProvider := ourProviderRaw.Meta.(*GraphNodeResourceProvider)
		ourProvider.Provider.Parent = pn.Provider
		ourProvider.Provider.parentNoun = n
	}

	g.Nouns = append(g.Nouns, nounsList...)
}

// graphAddConfigProviderConfigs adds a GraphNodeResourceProvider for every
// `provider` configuration block. Note that a provider may exist that
// isn't used for any resources. These will be pruned later.
func graphAddConfigProviderConfigs(g *depgraph.Graph, c *config.Config) {
	nounsList := make([]*depgraph.Noun, 0, len(c.ProviderConfigs))
	for _, pc := range c.ProviderConfigs {
		noun := &depgraph.Noun{
			Name: fmt.Sprintf("provider.%s", pc.Name),
			Meta: &GraphNodeResourceProvider{
				ID: pc.Name,
				Provider: &graphSharedProvider{
					Config: pc,
				},
			},
		}

		nounsList = append(nounsList, noun)
	}

	// Add all the provider config nouns to the graph
	g.Nouns = append(g.Nouns, nounsList...)
}

// graphAddRoot adds a root element to the graph so that there is a single
// root to point to all the dependencies.
func graphAddRoot(g *depgraph.Graph) {
	root := &depgraph.Noun{Name: GraphRootNode}
	for _, n := range g.Nouns {
		switch m := n.Meta.(type) {
		case *GraphNodeResource:
			// If the resource is part of a group, we don't need to make a dep
			if m.Index != -1 {
				continue
			}
		case *GraphNodeResourceMeta:
			// Always in the graph
		case *GraphNodeResourceProvider:
			// ResourceProviders don't need to be in the root deps because
			// they're always pointed to by some resource.
			continue
		}

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
		switch m := n.Meta.(type) {
		case *GraphNodeModule:
			if m.Config != nil {
				vars := m.Config.RawConfig.Variables
				nounAddVariableDeps(g, n, vars, false)
			}

		case *GraphNodeResource:
			if m.Config != nil {
				// Handle the resource variables
				vars := m.Config.RawConfig.Variables
				nounAddVariableDeps(g, n, vars, false)
			}

			// Handle the variables of the resource provisioners
			for _, p := range m.Resource.Provisioners {
				vars := p.RawConfig.Variables
				nounAddVariableDeps(g, n, vars, true)

				vars = p.ConnInfo.Variables
				nounAddVariableDeps(g, n, vars, true)
			}

		case *GraphNodeResourceProvider:
			if m.Provider != nil && m.Provider.Config != nil {
				vars := m.Provider.Config.RawConfig.Variables
				nounAddVariableDeps(g, n, vars, false)
			}

		default:
			// Other node types don't have dependencies or we don't support it
			continue
		}
	}
}

// graphAddTainted adds the tainted instances to the graph.
func graphAddTainted(g *depgraph.Graph, mod *ModuleState) {
	meta := g.Meta.(*GraphMeta)

	var nlist []*depgraph.Noun
	for k, rs := range mod.Resources {
		// If we have no tainted resources, continue on
		if len(rs.Tainted) == 0 {
			continue
		}

		// Find the untainted resource of this in the noun list
		var untainted *depgraph.Noun
		for _, n := range g.Nouns {
			if n.Name == k {
				untainted = n
				break
			}
		}

		for i, is := range rs.Tainted {
			name := fmt.Sprintf("%s (tainted #%d)", k, i+1)

			// Add each of the tainted resources to the graph, and encode
			// a dependency from the non-tainted resource to this so that
			// tainted resources are always destroyed first.
			noun := &depgraph.Noun{
				Name: name,
				Meta: &GraphNodeResource{
					Index: -1,
					Resource: &Resource{
						Id: k,
						Info: &InstanceInfo{
							Id:         k,
							ModulePath: meta.ModulePath,
							Type:       rs.Type,
						},
						State:        is,
						Config:       NewResourceConfig(nil),
						Diff:         &InstanceDiff{Destroy: true},
						Flags:        FlagTainted,
						TaintedIndex: i,
					},
				},
			}

			// Append it to the list so we handle it later
			nlist = append(nlist, noun)

			// If we have an untainted version, then make sure to add
			// the dependency.
			if untainted != nil {
				dep := &depgraph.Dependency{
					Name:   name,
					Source: untainted,
					Target: noun,
				}

				untainted.Deps = append(untainted.Deps, dep)
			}
		}
	}

	// Add the nouns to the graph
	g.Nouns = append(g.Nouns, nlist...)
}

// graphModuleNoun creates a noun for a module.
func graphModuleNoun(
	n string, m *config.Module,
	g *depgraph.Graph, opts *GraphOpts) (*depgraph.Noun, error) {
	name := fmt.Sprintf("module.%s", n)
	path := make([]string, len(opts.ModulePath)+1)
	copy(path, opts.ModulePath)
	path[len(opts.ModulePath)] = n

	// Build the opts we'll use to make the next graph
	subOpts := *opts
	subOpts.ModulePath = path
	subOpts.parent = g
	subGraph, err := Graph(&subOpts)
	if err != nil {
		return nil, fmt.Errorf(
			"Error building module graph '%s': %s",
			n, err)
	}

	return &depgraph.Noun{
		Name: name,
		Meta: &GraphNodeModule{
			Config: m,
			Path:   path,
			Graph:  subGraph,
		},
	}, nil
}

// nounAddVariableDeps updates the dependencies of a noun given
// a set of associated variable values
func nounAddVariableDeps(
	g *depgraph.Graph,
	n *depgraph.Noun,
	vars map[string]config.InterpolatedVariable,
	removeSelf bool) {
	for _, rawV := range vars {
		var name string
		var target *depgraph.Noun

		switch v := rawV.(type) {
		case *config.ModuleVariable:
			name = fmt.Sprintf("module.%s", v.Name)
			target = g.Noun(name)
		case *config.ResourceVariable:
			name = v.ResourceId()
			target = g.Noun(v.ResourceId())
		default:
		}

		if target == nil {
			continue
		}

		// If we're ignoring self-references, then don't add that
		// dependency.
		if removeSelf && n == target {
			continue
		}

		// Build the dependency
		dep := &depgraph.Dependency{
			Name:   name,
			Source: n,
			Target: target,
		}

		n.Deps = append(n.Deps, dep)
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

		sharedProvider := rn.Provider

		// Go through each prefix and instantiate if necessary, then
		// verify if this provider is of use to us or not.
		sharedProvider.Providers = make(map[string]ResourceProvider)
		sharedProvider.ProviderKeys = prefixes
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

			sharedProvider.Providers[prefix] = p
		}

		// If we never found a provider, then error and continue
		if len(sharedProvider.Providers) == 0 {
			errs = append(errs, fmt.Errorf(
				"Provider for configuration '%s' not found.",
				rn.ID))
			continue
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

// graphAddResourceProviderDeps goes through all the nodes in the graph
// and adds any dependencies to resource providers as needed.
func graphAddResourceProviderDeps(g *depgraph.Graph) {
	for _, rawN := range g.Nouns {
		switch n := rawN.Meta.(type) {
		case *GraphNodeModule:
			// Check if the module depends on any of our providers
			// by seeing if there is a parent node back.
			for _, moduleRaw := range n.Graph.Nouns {
				pn, ok := moduleRaw.Meta.(*GraphNodeResourceProvider)
				if !ok {
					continue
				}
				if pn.Provider.parentNoun == nil {
					continue
				}

				// Create the dependency to the provider
				dep := &depgraph.Dependency{
					Name:   pn.Provider.parentNoun.Name,
					Source: rawN,
					Target: pn.Provider.parentNoun,
				}
				rawN.Deps = append(rawN.Deps, dep)
			}
		case *GraphNodeResource:
			// Not sure how this would happen, but we might as well
			// check for it.
			if n.ResourceProviderNode == "" {
				continue
			}

			// Get the noun this depends on.
			target := g.Noun(n.ResourceProviderNode)

			// Create the dependency to the provider
			dep := &depgraph.Dependency{
				Name:   target.Name,
				Source: rawN,
				Target: target,
			}
			rawN.Deps = append(rawN.Deps, dep)
		}
	}
}

// graphPruneResourceProviders will remove the GraphNodeResourceProvider
// nodes that aren't used in any way.
func graphPruneResourceProviders(g *depgraph.Graph) {
	// First, build a mapping of the providers we have.
	ps := make(map[string]struct{})
	for _, n := range g.Nouns {
		_, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}

		ps[n.Name] = struct{}{}
	}

	// Now go through all the dependencies throughout and find
	// if any of these aren't reachable.
	for _, n := range g.Nouns {
		for _, dep := range n.Deps {
			delete(ps, dep.Target.Name)
		}
	}

	if len(ps) == 0 {
		// We used all of them!
		return
	}

	// Now go through and remove these nodes that aren't used
	for i := 0; i < len(g.Nouns); i++ {
		if _, ok := ps[g.Nouns[i].Name]; !ok {
			continue
		}

		// Delete this node
		copy(g.Nouns[i:], g.Nouns[i+1:])
		g.Nouns[len(g.Nouns)-1] = nil
		g.Nouns = g.Nouns[:len(g.Nouns)-1]
		i--
	}
}

// graphMapResourceProviderId goes through the graph and maps the
// ID of a resource provider node to each resource. This lets us know which
// configuration is for which resource.
//
// This is safe to call multiple times.
func graphMapResourceProviderId(g *depgraph.Graph) {
	// Build the list of provider configs we have
	ps := make(map[string]string)
	for _, n := range g.Nouns {
		pn, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}

		ps[n.Name] = pn.ID
	}

	// Go through every resource and find the shortest matching provider
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}

		var match, matchNode string
		for n, p := range ps {
			if !strings.HasPrefix(rn.Resource.Info.Type, p) {
				continue
			}
			if len(p) > len(match) {
				match = p
				matchNode = n
			}
		}
		if matchNode == "" {
			continue
		}

		rn.ResourceProviderNode = matchNode
	}
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

		rpnRaw := g.Noun(rn.ResourceProviderNode)
		if rpnRaw == nil {
			// This should never happen since when building the graph
			// we ensure that everything matches up.
			panic(fmt.Sprintf(
				"Resource provider not found: %s (type: %s)",
				rn.ResourceProviderNode,
				rn.Resource.Info.Type))
		}
		rpn := rpnRaw.Meta.(*GraphNodeResourceProvider)

		var provider ResourceProvider
		for _, k := range rpn.Provider.ProviderKeys {
			// Only try this provider if it has the right prefix
			if !strings.HasPrefix(rn.Resource.Info.Type, k) {
				continue
			}

			rp := rpn.Provider.Providers[k]
			if ProviderSatisfies(rp, rn.Resource.Info.Type) {
				provider = rp
				break
			}
		}

		if provider == nil {
			errs = append(errs, fmt.Errorf(
				"Resource provider not found for resource type '%s'",
				rn.Resource.Info.Type))
			continue
		}

		rn.Resource.Provider = provider
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

// graphMapResourceProvisioners takes a graph that already has
// the resources and maps the resource provisioners to the resources themselves.
func graphMapResourceProvisioners(g *depgraph.Graph,
	provisioners map[string]ResourceProvisionerFactory) error {
	var errs []error

	// Create a cache of resource provisioners, avoids duplicate
	// initialization of the instances
	cache := make(map[string]ResourceProvisioner)

	// Go through each of the resources and find a matching provisioners
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}

		// Ignore orphan nodes with no provisioners
		if rn.Config == nil {
			continue
		}

		// Check each provisioner
		for _, p := range rn.Config.Provisioners {
			// Check for a cached provisioner
			provisioner, ok := cache[p.Type]
			if !ok {
				// Lookup the factory method
				factory, ok := provisioners[p.Type]
				if !ok {
					errs = append(errs, fmt.Errorf(
						"Resource provisioner not found for provisioner type '%s'",
						p.Type))
					continue
				}

				// Initialize the provisioner
				prov, err := factory()
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"Failed to instantiate provisioner type '%s': %v",
						p.Type, err))
					continue
				}
				provisioner = prov

				// Cache this type of provisioner
				cache[p.Type] = prov
			}

			// Save the provisioner
			rn.Resource.Provisioners = append(rn.Resource.Provisioners, &ResourceProvisionerConfig{
				Type:        p.Type,
				Provisioner: provisioner,
				Config:      NewResourceConfig(p.RawConfig),
				RawConfig:   p.RawConfig,
				ConnInfo:    p.ConnInfo,
			})
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
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
