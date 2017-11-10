package terraform

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

func TransformProviders(providers []string, concrete ConcreteProviderNodeFunc, mod *module.Tree) GraphTransformer {
	return GraphTransformMulti(
		// Add providers from the config
		&ProviderConfigTransformer{
			Module:    mod,
			Providers: providers,
			Concrete:  concrete,
		},
		// Add any remaining missing providers
		&MissingProviderTransformer{
			Providers: providers,
			Concrete:  concrete,
		},
		// Connect the providers
		&ProviderTransformer{},
		// Remove unused providers and proxies
		&PruneProviderTransformer{},
		// Connect provider to their parent provider nodes
		&ParentProviderTransformer{},
	)
}

// GraphNodeProvider is an interface that nodes that can be a provider
// must implement.
// ProviderName returns the name of the provider this satisfies.
// Name returns the full name of the provider in the config.
type GraphNodeProvider interface {
	ProviderName() string
	Name() string
}

// GraphNodeCloseProvider is an interface that nodes that can be a close
// provider must implement. The CloseProviderName returned is the name of
// the provider they satisfy.
type GraphNodeCloseProvider interface {
	CloseProviderName() string
}

// GraphNodeProviderConsumer is an interface that nodes that require
// a provider must implement. ProvidedBy must return the name of the provider
// to use. This may be a provider by type, type.alias or a fully resolved
// provider name
type GraphNodeProviderConsumer interface {
	ProvidedBy() string
	// Set the resolved provider address for this resource.
	SetProvider(string)
}

// ProviderTransformer is a GraphTransformer that maps resources to
// providers within the graph. This will error if there are any resources
// that don't map to proper resources.
type ProviderTransformer struct{}

func (t *ProviderTransformer) Transform(g *Graph) error {
	// Go through the other nodes and match them to providers they need
	var err error
	m := providerVertexMap(g)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProviderConsumer); ok {
			p := pv.ProvidedBy()

			key := providerMapKey(p, pv)
			target := m[key]

			sp, ok := pv.(GraphNodeSubPath)
			if !ok && target == nil {
				// no target, and no path to walk up
				err = multierror.Append(err, fmt.Errorf(
					"%s: provider %s couldn't be found",
					dag.VertexName(v), p))
				break
			}

			// if we don't have a provider at this level, walk up the path looking for one
			for i := 1; target == nil; i++ {
				path := normalizeModulePath(sp.Path())
				if len(path) < i {
					break
				}

				key = ResolveProviderName(p, path[:len(path)-i])
				target = m[key]
				if target != nil {
					break
				}
			}

			if target == nil {
				err = multierror.Append(err, fmt.Errorf(
					"%s: configuration for %s is not present; a provider configuration block is required for all operations",
					dag.VertexName(v), p,
				))
				break
			}

			// see if this in  an inherited provider
			if p, ok := target.(*graphNodeProxyProvider); ok {
				g.Remove(p)
				target = p.Target()
				key = target.(GraphNodeProvider).Name()
			}

			log.Printf("[DEBUG] resource %s using provider %s", dag.VertexName(pv), key)
			pv.SetProvider(key)
			g.Connect(dag.BasicEdge(v, target))
		}
	}

	return err
}

// CloseProviderTransformer is a GraphTransformer that adds nodes to the
// graph that will close open provider connections that aren't needed anymore.
// A provider connection is not needed anymore once all depended resources
// in the graph are evaluated.
type CloseProviderTransformer struct{}

func (t *CloseProviderTransformer) Transform(g *Graph) error {
	pm := providerVertexMap(g)
	cpm := make(map[string]*graphNodeCloseProvider)
	var err error

	for _, v := range pm {
		p := v.(GraphNodeProvider)

		// get the close provider of this type if we alread created it
		closer := cpm[p.ProviderName()]

		if closer == nil {
			// create a closer for this provider type
			closer = &graphNodeCloseProvider{ProviderNameValue: p.ProviderName()}
			g.Add(closer)
			cpm[p.ProviderName()] = closer
		}

		// Close node depends on the provider itself
		// this is added unconditionally, so it will connect to all instances
		// of the provider. Extra edges will be removed by transitive
		// reduction.
		g.Connect(dag.BasicEdge(closer, p))

		// connect all the provider's resources to the close node
		for _, s := range g.UpEdges(p).List() {
			if _, ok := s.(GraphNodeProviderConsumer); ok {
				g.Connect(dag.BasicEdge(closer, s))
			}
		}
	}

	return err
}

// MissingProviderTransformer is a GraphTransformer that adds nodes for all
// required providers into the graph. Specifically, it creates provider
// configuration nodes for all the providers that we support. These are pruned
// later during an optimization pass.
type MissingProviderTransformer struct {
	// Providers is the list of providers we support.
	Providers []string

	// Concrete, if set, overrides how the providers are made.
	Concrete ConcreteProviderNodeFunc
}

func (t *MissingProviderTransformer) Transform(g *Graph) error {
	// Initialize factory
	if t.Concrete == nil {
		t.Concrete = func(a *NodeAbstractProvider) dag.Vertex {
			return a
		}
	}

	var err error
	m := providerVertexMap(g)
	for _, v := range g.Vertices() {
		pv, ok := v.(GraphNodeProviderConsumer)
		if !ok {
			continue
		}

		p := pv.ProvidedBy()
		key := ResolveProviderName(p, nil)
		provider := m[key]

		// we already have it
		if provider != nil {
			continue
		}

		// we don't implicitly create aliased providers
		if strings.Contains(p, ".") {
			log.Println("[DEBUG] not adding missing provider alias", p)
			continue
		}

		// create the misisng top-level provider
		provider = t.Concrete(&NodeAbstractProvider{
			NameValue: p,
		}).(dag.Vertex)

		m[key] = g.Add(provider)
	}

	return err
}

// ParentProviderTransformer connects provider nodes to their parents.
//
// This works by finding nodes that are both GraphNodeProviders and
// GraphNodeSubPath. It then connects the providers to their parent
// path. The parent provider is always at the root level.
type ParentProviderTransformer struct{}

func (t *ParentProviderTransformer) Transform(g *Graph) error {
	pm := providerVertexMap(g)
	for _, v := range g.Vertices() {
		// Only care about providers
		pn, ok := v.(GraphNodeProvider)
		if !ok || pn.ProviderName() == "" {
			continue
		}

		// Also require a subpath, if there is no subpath then we
		// can't have a parent.
		if pn, ok := v.(GraphNodeSubPath); ok {
			if len(normalizeModulePath(pn.Path())) <= 1 {
				continue
			}
		}

		// this provider may be disabled, but we can only get it's name from
		// the ProviderName string
		name := ResolveProviderName(strings.SplitN(pn.ProviderName(), " ", 2)[0], nil)
		parent := pm[name]
		if parent != nil {
			g.Connect(dag.BasicEdge(v, parent))
		}

	}
	return nil
}

// PruneProviderTransformer removes any providers that are not actually used by
// anything, and provider proxies. This avoids the provider being initialized
// and configured.  This both saves resources but also avoids errors since
// configuration may imply initialization which may require auth.
type PruneProviderTransformer struct{}

func (t *PruneProviderTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about providers
		pn, ok := v.(GraphNodeProvider)
		if !ok || pn.ProviderName() == "" {
			continue
		}

		// ProxyProviders will have up edges, but we're now done with them in the graph
		if _, ok := v.(*graphNodeProxyProvider); ok {
			log.Printf("[DEBUG] pruning proxy provider %s", dag.VertexName(v))
			g.Remove(v)
		}

		// Remove providers with no dependencies.
		if g.UpEdges(v).Len() == 0 {
			log.Printf("[DEBUG] pruning unused provider %s", dag.VertexName(v))
			g.Remove(v)
		}
	}

	return nil
}

// providerMapKey is a helper that gives us the key to use for the
// maps returned by things such as providerVertexMap.
func providerMapKey(k string, v dag.Vertex) string {
	if strings.Contains(k, "provider.") {
		// this is already resolved
		return k
	}

	// we create a dummy provider to
	var path []string
	if sp, ok := v.(GraphNodeSubPath); ok {
		path = normalizeModulePath(sp.Path())
	}
	return ResolveProviderName(k, path)
}

func providerVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvider); ok {
			// TODO:  The Name may have meta info, like " (disabled)"
			name := strings.SplitN(pv.Name(), " ", 2)[0]
			m[name] = v
		}
	}

	return m
}

func closeProviderVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeCloseProvider); ok {
			m[pv.CloseProviderName()] = v
		}
	}

	return m
}

type graphNodeCloseProvider struct {
	ProviderNameValue string
}

func (n *graphNodeCloseProvider) Name() string {
	return fmt.Sprintf("provider.%s (close)", n.ProviderNameValue)
}

// GraphNodeEvalable impl.
func (n *graphNodeCloseProvider) EvalTree() EvalNode {
	return CloseProviderEvalTree(n.ProviderNameValue)
}

// GraphNodeDependable impl.
func (n *graphNodeCloseProvider) DependableName() []string {
	return []string{n.Name()}
}

func (n *graphNodeCloseProvider) CloseProviderName() string {
	return n.ProviderNameValue
}

// GraphNodeDotter impl.
func (n *graphNodeCloseProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	if !opts.Verbose {
		return nil
	}
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}

// RemovableIfNotTargeted
func (n *graphNodeCloseProvider) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// graphNodeProxyProvider is a GraphNodeProvider implementation that is used to
// store the name and value of a provider node for inheritance between modules.
// These nodes are only used to store the data while loading the provider
// configurations, and are removed after all the resources have been connected
// to their providers.
type graphNodeProxyProvider struct {
	nameValue string
	path      []string
	target    GraphNodeProvider
}

func (n *graphNodeProxyProvider) ProviderName() string {
	return n.Target().ProviderName()
}

func (n *graphNodeProxyProvider) Name() string {
	return ResolveProviderName(n.nameValue, n.path)
}

// find the concrete provider instance
func (n *graphNodeProxyProvider) Target() GraphNodeProvider {
	switch t := n.target.(type) {
	case *graphNodeProxyProvider:
		return t.Target()
	default:
		return n.target
	}
}

// ProviderConfigTransformer adds all provider nodes from the configuration and
// attaches the configs.
type ProviderConfigTransformer struct {
	Providers []string
	Concrete  ConcreteProviderNodeFunc

	// each provider node is stored here so that the proxy nodes can look up
	// their targets by name.
	providers map[string]GraphNodeProvider
	// record providers that can be overriden with a proxy
	proxiable map[string]bool

	// Module is the module to add resources from.
	Module *module.Tree
}

func (t *ProviderConfigTransformer) Transform(g *Graph) error {
	// If no module is given, we don't do anything
	if t.Module == nil {
		return nil
	}

	// If the module isn't loaded, that is simply an error
	if !t.Module.Loaded() {
		return errors.New("module must be loaded for ProviderConfigTransformer")
	}

	t.providers = make(map[string]GraphNodeProvider)
	t.proxiable = make(map[string]bool)

	// Start the transformation process
	if err := t.transform(g, t.Module); err != nil {
		return nil
	}

	// finally attach the configs to the new nodes
	return t.attachProviderConfigs(g)
}

func (t *ProviderConfigTransformer) transform(g *Graph, m *module.Tree) error {
	// If no config, do nothing
	if m == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, m); err != nil {
		return err
	}

	// Transform all the children.
	for _, c := range m.Children() {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}
	return nil
}

func (t *ProviderConfigTransformer) transformSingle(g *Graph, m *module.Tree) error {
	log.Printf("[TRACE] ProviderConfigTransformer: Starting for path: %v", m.Path())

	// Get the configuration for this module
	conf := m.Config()

	// Build the path we're at
	path := m.Path()
	if len(path) > 0 {
		path = append([]string{RootModuleName}, path...)
	}

	// add all providers from the configuration
	for _, p := range conf.ProviderConfigs {
		name := p.Name
		if p.Alias != "" {
			name += "." + p.Alias
		}

		v := t.Concrete(&NodeAbstractProvider{
			NameValue: name,
			PathValue: path,
		})

		// Add it to the graph
		g.Add(v)
		fullName := ResolveProviderName(name, path)
		t.providers[fullName] = v.(GraphNodeProvider)
		t.proxiable[fullName] = len(p.RawConfig.RawMap()) == 0
	}

	// Now replace the provider nodes with proxy nodes if a provider was being
	// passed in, and create implicit proxies if there was no config. Any extra
	// proxies will be removed in the prune step.
	return t.addProxyProviders(g, m)
}

func (t *ProviderConfigTransformer) addProxyProviders(g *Graph, m *module.Tree) error {
	path := m.Path()

	// can't add proxies at the root
	if len(path) == 0 {
		return nil
	}

	parentPath := path[:len(path)-1]
	parent := t.Module.Child(parentPath)
	if parent == nil {
		return nil
	}

	var parentCfg *config.Module
	for _, mod := range parent.Config().Modules {
		if mod.Name == m.Name() {
			parentCfg = mod
			break
		}
	}

	if parentCfg == nil {
		// this can't really happen during normal execution.
		return fmt.Errorf("parent module config not found for %s", m.Name())
	}

	// Go through all the providers the parent is passing in, and add proxies to
	// the parent provider nodes.
	for name, parentName := range parentCfg.Providers {
		fullName := ResolveProviderName(name, path)
		fullParentName := ResolveProviderName(parentName, parentPath)
		parentProvider := t.providers[fullParentName]

		if parentProvider == nil {
			return fmt.Errorf("missing provider %s", fullParentName)
		}

		proxy := &graphNodeProxyProvider{
			nameValue: name,
			path:      path,
			target:    parentProvider,
		}

		concreteProvider := t.providers[fullName]

		// replace the concrete node with the provider passed in
		if concreteProvider != nil && t.proxiable[fullName] {
			g.Replace(concreteProvider, proxy)
			t.providers[fullName] = proxy
			continue
		}

		// aliased providers can't be implicitly passed in
		if strings.Contains(name, ".") {
			continue
		}

		// There was no concrete provider, so add this as an implicit provider.
		// The extra proxy will be pruned later if it's unused.
		g.Add(proxy)
		t.providers[fullName] = proxy
	}
	return nil
}

func (t *ProviderConfigTransformer) attachProviderConfigs(g *Graph) error {
	for _, v := range g.Vertices() {
		// Only care about GraphNodeAttachProvider implementations
		apn, ok := v.(GraphNodeAttachProvider)
		if !ok {
			continue
		}

		// Determine what we're looking for
		path := normalizeModulePath(apn.Path())[1:]
		name := apn.ProviderName()
		log.Printf("[TRACE] Attach provider request: %#v %s", path, name)

		// Get the configuration.
		tree := t.Module.Child(path)
		if tree == nil {
			continue
		}

		// Go through the provider configs to find the matching config
		for _, p := range tree.Config().ProviderConfigs {
			// Build the name, which is "name.alias" if an alias exists
			current := p.Name
			if p.Alias != "" {
				current += "." + p.Alias
			}

			// If the configs match then attach!
			if current == name {
				log.Printf("[TRACE] Attaching provider config: %#v", p)
				apn.AttachProvider(p)
				break
			}
		}
	}

	return nil
}
