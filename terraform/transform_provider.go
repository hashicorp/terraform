package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

func TransformProviders(providers []string, concrete ConcreteProviderNodeFunc, config *configs.Config) GraphTransformer {
	return GraphTransformMulti(
		// Add providers from the config
		&ProviderConfigTransformer{
			Config:    config,
			Providers: providers,
			Concrete:  concrete,
		},
		// Add any remaining missing providers
		&MissingProviderTransformer{
			Providers: providers,
			Concrete:  concrete,
		},
		// Connect the providers
		&ProviderTransformer{
			Config: config,
		},
		// Remove unused providers and proxies
		&PruneProviderTransformer{},
		// Connect provider to their parent provider nodes
		&ParentProviderTransformer{},
	)
}

// GraphNodeProvider is an interface that nodes that can be a provider
// must implement.
//
// ProviderAddr returns the address of the provider configuration this
// satisfies, which is relative to the path returned by method Path().
//
// Name returns the full name of the provider in the config.
type GraphNodeProvider interface {
	GraphNodeSubPath
	ProviderAddr() addrs.AbsProviderConfig
	Name() string
}

// GraphNodeCloseProvider is an interface that nodes that can be a close
// provider must implement. The CloseProviderName returned is the name of
// the provider they satisfy.
type GraphNodeCloseProvider interface {
	GraphNodeSubPath
	CloseProviderAddr() addrs.AbsProviderConfig
}

// GraphNodeProviderConsumer is an interface that nodes that require
// a provider must implement. ProvidedBy must return the address of the provider
// to use, which will be resolved to a configuration either in the same module
// or in an ancestor module, with the resulting absolute address passed to
// SetProvider.
type GraphNodeProviderConsumer interface {
	// ProvidedBy returns the address of the provider configuration the node
	// refers to. If the returned "exact" value is true, this address will
	// be taken exactly. If "exact" is false, a provider configuration from
	// an ancestor module may be selected instead.
	ProvidedBy() (addr addrs.AbsProviderConfig, exact bool)
	// Set the resolved provider address for this resource.
	SetProvider(addrs.AbsProviderConfig)
}

// ProviderTransformer is a GraphTransformer that maps resources to
// providers within the graph. This will error if there are any resources
// that don't map to proper resources.
type ProviderTransformer struct {
	Config *configs.Config
}

func (t *ProviderTransformer) Transform(g *Graph) error {
	// We need to find a provider configuration address for each resource
	// either directly represented by a node or referenced by a node in
	// the graph, and then create graph edges from provider to provider user
	// so that the providers will get initialized first.

	var diags tfdiags.Diagnostics

	// To start, we'll collect the _requested_ provider addresses for each
	// node, which we'll then resolve (handling provider inheritence, etc) in
	// the next step.
	// Our "requested" map is from graph vertices to string representations of
	// provider config addresses (for deduping) to requests.
	type ProviderRequest struct {
		Addr  addrs.AbsProviderConfig
		Exact bool // If true, inheritence from parent modules is not attempted
	}
	requested := map[dag.Vertex]map[string]ProviderRequest{}
	needConfigured := map[string]addrs.AbsProviderConfig{}
	for _, v := range g.Vertices() {

		// Does the vertex _directly_ use a provider?
		if pv, ok := v.(GraphNodeProviderConsumer); ok {
			requested[v] = make(map[string]ProviderRequest)

			p, exact := pv.ProvidedBy()
			if exact {
				log.Printf("[TRACE] ProviderTransformer: %s is provided by %s exactly", dag.VertexName(v), p)
			} else {
				log.Printf("[TRACE] ProviderTransformer: %s is provided by %s or inherited equivalent", dag.VertexName(v), p)
			}

			requested[v][p.String()] = ProviderRequest{
				Addr:  p,
				Exact: exact,
			}

			// Direct references need the provider configured as well as initialized
			needConfigured[p.String()] = p
		}
	}

	// Now we'll go through all the requested addresses we just collected and
	// figure out which _actual_ config address each belongs to, after resolving
	// for provider inheritance and passing.
	m := providerVertexMap(g)
	for v, reqs := range requested {
		for key, req := range reqs {
			p := req.Addr
			target := m[key]

			_, ok := v.(GraphNodeSubPath)
			if !ok && target == nil {
				// No target and no path to traverse up from
				diags = diags.Append(fmt.Errorf("%s: provider %s couldn't be found", dag.VertexName(v), p))
				continue
			}

			if target != nil {
				log.Printf("[TRACE] ProviderTransformer: exact match for %s serving %s", p, dag.VertexName(v))
			}

			// if we don't have a provider at this level, walk up the path looking for one,
			// unless we were told to be exact.
			if target == nil && !req.Exact {
				for pp, ok := p.Inherited(); ok; pp, ok = pp.Inherited() {
					key := pp.String()
					target = m[key]
					if target != nil {
						log.Printf("[TRACE] ProviderTransformer: %s uses inherited configuration %s", dag.VertexName(v), pp)
						break
					}
					log.Printf("[TRACE] ProviderTransformer: looking for %s to serve %s", pp, dag.VertexName(v))
				}
			}

			// If this provider doesn't need to be configured then we can just
			// stub it out with an init-only provider node, which will just
			// start up the provider and fetch its schema.
			if _, exists := needConfigured[key]; target == nil && !exists {
				stubAddr := p.ProviderConfig.Absolute(addrs.RootModuleInstance)
				stub := &NodeEvalableProvider{
					&NodeAbstractProvider{
						Addr: stubAddr,
					},
				}
				m[stubAddr.String()] = stub
				log.Printf("[TRACE] ProviderTransformer: creating init-only node for %s", stubAddr)
				target = stub
				g.Add(target)
			}

			if target == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider configuration not present",
					fmt.Sprintf(
						"To work with %s its original provider configuration at %s is required, but it has been removed. This occurs when a provider configuration is removed while objects created by that provider still exist in the state. Re-add the provider configuration to destroy %s, after which you can remove the provider configuration again.",
						dag.VertexName(v), p, dag.VertexName(v),
					),
				))
				break
			}

			// see if this in  an inherited provider
			if p, ok := target.(*graphNodeProxyProvider); ok {
				g.Remove(p)
				target = p.Target()
				key = target.(GraphNodeProvider).ProviderAddr().String()
			}

			log.Printf("[DEBUG] ProviderTransformer: %q (%T) needs %s", dag.VertexName(v), v, dag.VertexName(target))
			if pv, ok := v.(GraphNodeProviderConsumer); ok {
				pv.SetProvider(target.ProviderAddr())
			}
			g.Connect(dag.BasicEdge(v, target))
		}
	}

	return diags.Err()
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
		key := p.ProviderAddr().String()

		// get the close provider of this type if we alread created it
		closer := cpm[key]

		if closer == nil {
			// create a closer for this provider type
			closer = &graphNodeCloseProvider{Addr: p.ProviderAddr()}
			g.Add(closer)
			cpm[key] = closer
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

// MissingProviderTransformer is a GraphTransformer that adds to the graph
// a node for each default provider configuration that is referenced by another
// node but not already present in the graph.
//
// These "default" nodes are always added to the root module, regardless of
// where they are requested. This is important because our inheritance
// resolution behavior in ProviderTransformer will then treat these as a
// last-ditch fallback after walking up the tree, rather than preferring them
// as it would if they were placed in the same module as the requester.
//
// This transformer may create extra nodes that are not needed in practice,
// due to overriding provider configurations in child modules.
// PruneProviderTransformer can then remove these once ProviderTransformer
// has resolved all of the inheritence, etc.
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

		// For our work here we actually care only about the provider type and
		// we plan to place all default providers in the root module, and so
		// it's safe for us to rely on ProvidedBy here rather than waiting for
		// the later proper resolution of provider inheritance done by
		// ProviderTransformer.
		p, _ := pv.ProvidedBy()
		if p.ProviderConfig.Alias != "" {
			// We do not create default aliased configurations.
			log.Println("[TRACE] MissingProviderTransformer: skipping implication of aliased config", p)
			continue
		}

		// We're going to create an implicit _default_ configuration for the
		// referenced provider type in the _root_ module, ignoring all other
		// aspects of the resource's declared provider address.
		defaultAddr := addrs.RootModuleInstance.ProviderConfigDefault(p.ProviderConfig.Type)
		key := defaultAddr.String()
		provider := m[key]

		if provider != nil {
			// There's already an explicit default configuration for this
			// provider type in the root module, so we have nothing to do.
			continue
		}

		log.Printf("[DEBUG] adding implicit provider configuration %s, implied first by %s", defaultAddr, dag.VertexName(v))

		// create the missing top-level provider
		provider = t.Concrete(&NodeAbstractProvider{
			Addr: defaultAddr,
		}).(GraphNodeProvider)

		g.Add(provider)
		m[key] = provider
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
		if !ok {
			continue
		}

		// Also require non-empty path, since otherwise we're in the root
		// module and so cannot have a parent.
		if len(pn.Path()) <= 1 {
			continue
		}

		// this provider may be disabled, but we can only get it's name from
		// the ProviderName string
		addr := pn.ProviderAddr()
		parentAddr, ok := addr.Inherited()
		if ok {
			parent := pm[parentAddr.String()]
			if parent != nil {
				g.Connect(dag.BasicEdge(v, parent))
			}
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
		_, ok := v.(GraphNodeProvider)
		if !ok {
			continue
		}

		// ProxyProviders will have up edges, but we're now done with them in the graph
		if _, ok := v.(*graphNodeProxyProvider); ok {
			log.Printf("[DEBUG] pruning proxy %s", dag.VertexName(v))
			g.Remove(v)
		}

		// Remove providers with no dependencies.
		if g.UpEdges(v).Len() == 0 {
			log.Printf("[DEBUG] pruning unused %s", dag.VertexName(v))
			g.Remove(v)
		}
	}

	return nil
}

func providerVertexMap(g *Graph) map[string]GraphNodeProvider {
	m := make(map[string]GraphNodeProvider)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvider); ok {
			addr := pv.ProviderAddr()
			m[addr.String()] = pv
		}
	}

	return m
}

func closeProviderVertexMap(g *Graph) map[string]GraphNodeCloseProvider {
	m := make(map[string]GraphNodeCloseProvider)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeCloseProvider); ok {
			addr := pv.CloseProviderAddr()
			m[addr.String()] = pv
		}
	}

	return m
}

type graphNodeCloseProvider struct {
	Addr addrs.AbsProviderConfig
}

var (
	_ GraphNodeCloseProvider = (*graphNodeCloseProvider)(nil)
)

func (n *graphNodeCloseProvider) Name() string {
	return n.Addr.String() + " (close)"
}

// GraphNodeSubPath impl.
func (n *graphNodeCloseProvider) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeEvalable impl.
func (n *graphNodeCloseProvider) EvalTree() EvalNode {
	return CloseProviderEvalTree(n.Addr)
}

// GraphNodeDependable impl.
func (n *graphNodeCloseProvider) DependableName() []string {
	return []string{n.Name()}
}

func (n *graphNodeCloseProvider) CloseProviderAddr() addrs.AbsProviderConfig {
	return n.Addr
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
	addr   addrs.AbsProviderConfig
	target GraphNodeProvider
}

var (
	_ GraphNodeProvider = (*graphNodeProxyProvider)(nil)
)

func (n *graphNodeProxyProvider) ProviderAddr() addrs.AbsProviderConfig {
	return n.addr
}

func (n *graphNodeProxyProvider) Path() addrs.ModuleInstance {
	return n.addr.Module
}

func (n *graphNodeProxyProvider) Name() string {
	return n.addr.String() + " (proxy)"
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

	// Config is the root node of the configuration tree to add providers from.
	Config *configs.Config
}

func (t *ProviderConfigTransformer) Transform(g *Graph) error {
	// If no configuration is given, we don't do anything
	if t.Config == nil {
		return nil
	}

	t.providers = make(map[string]GraphNodeProvider)
	t.proxiable = make(map[string]bool)

	// Start the transformation process
	if err := t.transform(g, t.Config); err != nil {
		return err
	}

	// finally attach the configs to the new nodes
	return t.attachProviderConfigs(g)
}

func (t *ProviderConfigTransformer) transform(g *Graph, c *configs.Config) error {
	// If no config, do nothing
	if c == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, c); err != nil {
		return err
	}

	// Transform all the children.
	for _, cc := range c.Children {
		if err := t.transform(g, cc); err != nil {
			return err
		}
	}
	return nil
}

func (t *ProviderConfigTransformer) transformSingle(g *Graph, c *configs.Config) error {
	// Get the module associated with this configuration tree node
	mod := c.Module
	staticPath := c.Path

	// We actually need a dynamic module path here, but we've not yet updated
	// our graph builders enough to support expansion of module calls with
	// "count" and "for_each" set, so for now we'll shim this by converting to
	// a dynamic path with no keys. At the time of writing this is the only
	// possible kind of dynamic path anyway.
	path := make(addrs.ModuleInstance, len(staticPath))
	for i, name := range staticPath {
		path[i] = addrs.ModuleInstanceStep{
			Name: name,
		}
	}

	// add all providers from the configuration
	for _, p := range mod.ProviderConfigs {
		relAddr := p.Addr()
		addr := relAddr.Absolute(path)

		abstract := &NodeAbstractProvider{
			Addr: addr,
		}
		var v dag.Vertex
		if t.Concrete != nil {
			v = t.Concrete(abstract)
		} else {
			v = abstract
		}

		// Add it to the graph
		g.Add(v)
		key := addr.String()
		t.providers[key] = v.(GraphNodeProvider)

		// A provider configuration is "proxyable" if its configuration is
		// entirely empty. This means it's standing in for a provider
		// configuration that must be passed in from the parent module.
		// We decide this by evaluating the config with an empty schema;
		// if this succeeds, then we know there's nothing in the body.
		_, diags := p.Config.Content(&hcl.BodySchema{})
		t.proxiable[key] = !diags.HasErrors()
	}

	// Now replace the provider nodes with proxy nodes if a provider was being
	// passed in, and create implicit proxies if there was no config. Any extra
	// proxies will be removed in the prune step.
	return t.addProxyProviders(g, c)
}

func (t *ProviderConfigTransformer) addProxyProviders(g *Graph, c *configs.Config) error {
	path := c.Path

	// can't add proxies at the root
	if len(path) == 0 {
		return nil
	}

	parentPath, callAddr := path.Call()
	parent := c.Parent
	if parent == nil {
		return nil
	}

	callName := callAddr.Name
	var parentCfg *configs.ModuleCall
	for name, mod := range parent.Module.ModuleCalls {
		if name == callName {
			parentCfg = mod
			break
		}
	}

	// We currently don't support count/for_each for modules and so we must
	// shim our path and parentPath into module instances here so that the
	// rest of Terraform can behave as if we do. This shimming should be
	// removed later as part of implementing count/for_each for modules.
	instPath := make(addrs.ModuleInstance, len(path))
	for i, name := range path {
		instPath[i] = addrs.ModuleInstanceStep{Name: name}
	}
	parentInstPath := make(addrs.ModuleInstance, len(parentPath))
	for i, name := range parentPath {
		parentInstPath[i] = addrs.ModuleInstanceStep{Name: name}
	}

	if parentCfg == nil {
		// this can't really happen during normal execution.
		return fmt.Errorf("parent module config not found for %s", c.Path.String())
	}

	// Go through all the providers the parent is passing in, and add proxies to
	// the parent provider nodes.
	for _, pair := range parentCfg.Providers {
		fullAddr := pair.InChild.Addr().Absolute(instPath)
		fullParentAddr := pair.InParent.Addr().Absolute(parentInstPath)
		fullName := fullAddr.String()
		fullParentName := fullParentAddr.String()

		parentProvider := t.providers[fullParentName]

		if parentProvider == nil {
			return fmt.Errorf("missing provider %s", fullParentName)
		}

		proxy := &graphNodeProxyProvider{
			addr:   fullAddr,
			target: parentProvider,
		}

		concreteProvider := t.providers[fullName]

		// replace the concrete node with the provider passed in
		if concreteProvider != nil && t.proxiable[fullName] {
			g.Replace(concreteProvider, proxy)
			t.providers[fullName] = proxy
			continue
		}

		// aliased configurations can't be implicitly passed in
		if fullAddr.ProviderConfig.Alias != "" {
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
		addr := apn.ProviderAddr()

		// Get the configuration.
		mc := t.Config.DescendentForInstance(addr.Module)
		if mc == nil {
			log.Printf("[TRACE] ProviderConfigTransformer: no configuration available for %s", addr.String())
			continue
		}

		// Go through the provider configs to find the matching config
		for _, p := range mc.Module.ProviderConfigs {
			if p.Name == addr.ProviderConfig.Type && p.Alias == addr.ProviderConfig.Alias {
				log.Printf("[TRACE] ProviderConfigTransformer: attaching to %q provider configuration from %s", dag.VertexName(v), p.DeclRange)
				apn.AttachProvider(p)
				break
			}
		}
	}

	return nil
}
