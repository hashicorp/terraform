// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func transformProviders(concrete ConcreteProviderNodeFunc, config *configs.Config, externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface) GraphTransformer {
	return GraphTransformMulti(
		// Add placeholder nodes for any externally-configured providers
		&externalProviderTransformer{
			ExternalProviderConfigs: externalProviderConfigs,
		},
		// Add providers from the config
		&ProviderConfigTransformer{
			Config:   config,
			Concrete: concrete,
		},
		// Add any remaining missing providers
		&MissingProviderTransformer{
			Config:   config,
			Concrete: concrete,
		},
		// Connect the providers
		&ProviderTransformer{
			Config: config,
		},
		// Remove unused providers and proxies
		&PruneProviderTransformer{},
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
	GraphNodeModulePath
	ProviderAddr() addrs.AbsProviderConfig
	Name() string
}

// GraphNodeCloseProvider is an interface that nodes that can be a close
// provider must implement. The CloseProviderName returned is the name of
// the provider they satisfy.
type GraphNodeCloseProvider interface {
	GraphNodeModulePath
	CloseProviderAddr() addrs.AbsProviderConfig
}

// GraphNodeProviderConsumer is an interface that nodes that require
// a provider must implement. ProvidedBy must return the address of the provider
// to use, which will be resolved to a configuration either in the same module
// or in an ancestor module, with the resulting absolute address passed to
// SetProvider.
type GraphNodeProviderConsumer interface {
	GraphNodeModulePath
	// ProvidedBy returns the address of the provider configuration the node
	// refers to, if available. The following value types may be returned:
	//
	//   nil + exact true: the node does not require a provider
	// * addrs.LocalProviderConfig: the provider was set in the resource config
	// * addrs.AbsProviderConfig + exact true: the provider configuration was
	//   taken from the instance state.
	// * addrs.AbsProviderConfig + exact false: no config or state; the returned
	//   value is a default provider configuration address for the resource's
	//   Provider
	ProvidedBy() (addr addrs.ProviderConfig, exact bool)

	// Provider() returns the Provider FQN for the node.
	Provider() (provider addrs.Provider)

	// Set the resolved provider address for this resource.
	SetProvider(addrs.AbsProviderConfig)
}

// ProviderTransformer is a GraphTransformer that maps resources to providers
// within the graph. This will error if there are any resources that don't map
// to proper resources.
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
			providerAddr, exact := pv.ProvidedBy()
			if providerAddr == nil && exact {
				// no provider is required
				continue
			}

			requested[v] = make(map[string]ProviderRequest)

			var absPc addrs.AbsProviderConfig

			switch p := providerAddr.(type) {
			case addrs.AbsProviderConfig:
				// ProvidedBy() returns an AbsProviderConfig when the provider
				// configuration is set in state, so we do not need to verify
				// the FQN matches.
				absPc = p

				if exact {
					log.Printf("[TRACE] ProviderTransformer: %s is provided by %s exactly", dag.VertexName(v), absPc)
				}

			case addrs.LocalProviderConfig:
				// ProvidedBy() return a LocalProviderConfig when the resource
				// contains a `provider` attribute
				absPc.Provider = pv.Provider()
				modPath := pv.ModulePath()
				if t.Config == nil {
					absPc.Module = modPath
					absPc.Alias = p.Alias
					break
				}

				absPc.Module = modPath
				absPc.Alias = p.Alias

			default:
				// This should never happen; the case statements are meant to be exhaustive
				panic(fmt.Sprintf("%s: provider for %s couldn't be determined", dag.VertexName(v), absPc))
			}

			requested[v][absPc.String()] = ProviderRequest{
				Addr:  absPc,
				Exact: exact,
			}

			// Direct references need the provider configured as well as initialized
			needConfigured[absPc.String()] = absPc
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

			_, ok := v.(GraphNodeModulePath)
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
				stubAddr := addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: p.Provider,
				}
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

			// see if this is a proxy provider pointing to another concrete config
			if p, ok := target.(*graphNodeProxyProvider); ok {
				g.Remove(p)
				target = p.Target()
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

	for _, p := range pm {
		key := p.ProviderAddr().String()

		// make sure we haven't created the closer node already
		_, ok := cpm[key]
		if ok {
			log.Printf("[ERROR] CloseProviderTransformer: already created close node for %s", key)
			continue
		}

		// create a closer for this provider type
		closer := &graphNodeCloseProvider{Addr: p.ProviderAddr()}
		g.Add(closer)
		cpm[key] = closer

		// Close node depends on the provider itself
		// this is added unconditionally, so it will connect to all instances
		// of the provider. Extra edges will be removed by transitive
		// reduction.
		g.Connect(dag.BasicEdge(closer, p))
	}

	// Now look for all provider consumers and connect them to the appropriate closers.
	for _, v := range g.Vertices() {
		pc, ok := v.(GraphNodeProviderConsumer)
		if !ok {
			continue
		}

		p, exact := pc.ProvidedBy()
		if p == nil && exact {
			// this node does not require a provider
			continue
		}

		provider, ok := p.(addrs.AbsProviderConfig)
		if !ok {
			return fmt.Errorf("%s failed to return a provider reference", dag.VertexName(pc))
		}

		closer, ok := cpm[provider.String()]
		if !ok {
			return fmt.Errorf("no graphNodeCloseProvider for %s", provider)
		}
		g.Connect(dag.BasicEdge(closer, v))
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
	// MissingProviderTransformer needs the config to rule out _implied_ default providers
	Config *configs.Config

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
		// we plan to place all default providers in the root module.
		providerFqn := pv.Provider()

		// We're going to create an implicit _default_ configuration for the
		// referenced provider type in the _root_ module, ignoring all other
		// aspects of the resource's declared provider address.
		defaultAddr := addrs.RootModuleInstance.ProviderConfigDefault(providerFqn)
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

type graphNodeCloseProvider struct {
	Addr addrs.AbsProviderConfig
}

var (
	_ GraphNodeCloseProvider = (*graphNodeCloseProvider)(nil)
	_ GraphNodeExecutable    = (*graphNodeCloseProvider)(nil)
)

func (n *graphNodeCloseProvider) Name() string {
	return n.Addr.String() + " (close)"
}

// GraphNodeModulePath
func (n *graphNodeCloseProvider) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeExecutable impl.
func (n *graphNodeCloseProvider) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	return diags.Append(ctx.CloseProvider(n.Addr))
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
	_ GraphNodeModulePath = (*graphNodeProxyProvider)(nil)
	_ GraphNodeProvider   = (*graphNodeProxyProvider)(nil)
)

func (n *graphNodeProxyProvider) ProviderAddr() addrs.AbsProviderConfig {
	return n.addr
}

func (n *graphNodeProxyProvider) ModulePath() addrs.Module {
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
	Concrete ConcreteProviderNodeFunc

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

	// We'll start with any provider nodes that are already in the graph,
	// just so we can avoid creating any duplicates.
	t.providers = providerVertexMap(g)
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
	path := c.Path

	// If this is the root module, we can add nodes for required providers that
	// have no configuration, equivalent to having an empty configuration
	// block. This will ensure that a provider node exists for modules to
	// access when passing around configuration and inheritance.
	if path.IsRoot() && c.Module.ProviderRequirements != nil {
		for name, p := range c.Module.ProviderRequirements.RequiredProviders {
			if _, configured := mod.ProviderConfigs[name]; configured {
				continue
			}

			addr := addrs.AbsProviderConfig{
				Provider: p.Type,
				Module:   path,
			}

			if _, ok := t.providers[addr.String()]; ok {
				// The config validation warns about this too, but we can't
				// completely prevent it in v1.
				log.Printf("[WARN] ProviderConfigTransformer: duplicate required_providers entry for %s", addr)
				continue
			}

			abstract := &NodeAbstractProvider{
				Addr: addr,
			}

			var v dag.Vertex
			if t.Concrete != nil {
				v = t.Concrete(abstract)
			} else {
				v = abstract
			}

			g.Add(v)
			t.providers[addr.String()] = v.(GraphNodeProvider)
		}
	}

	// add all providers from the configuration
	for _, p := range mod.ProviderConfigs {
		fqn := mod.ProviderForLocalConfig(p.Addr())
		addr := addrs.AbsProviderConfig{
			Provider: fqn,
			Alias:    p.Alias,
			Module:   path,
		}

		if _, ok := t.providers[addr.String()]; ok {
			// The abstract provider node may already have been added from the
			// provider requirements.
			log.Printf("[WARN] ProviderConfigTransformer: provider node %s already added", addr)
			continue
		}

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

		// While deprecated, we still accept empty configuration blocks within
		// modules as being a possible proxy for passed configuration.
		if !path.IsRoot() {
			// A provider configuration is "proxyable" if its configuration is
			// entirely empty. This means it's standing in for a provider
			// configuration that must be passed in from the parent module.
			// We decide this by evaluating the config with an empty schema;
			// if this succeeds, then we know there's nothing in the body.
			_, diags := p.Config.Content(&hcl.BodySchema{})
			t.proxiable[key] = !diags.HasErrors()
		}
	}

	// Now replace the provider nodes with proxy nodes if a provider was being
	// passed in, and create implicit proxies if there was no config. Any extra
	// proxies will be removed in the prune step.
	return t.addProxyProviders(g, c)
}

func (t *ProviderConfigTransformer) addProxyProviders(g *Graph, c *configs.Config) error {
	path := c.Path

	// can't add proxies at the root
	if path.IsRoot() {
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

	if parentCfg == nil {
		// this can't really happen during normal execution.
		return fmt.Errorf("parent module config not found for %s", c.Path.String())
	}

	// Go through all the providers the parent is passing in, and add proxies to
	// the parent provider nodes.
	for _, pair := range parentCfg.Providers {
		fqn := c.Module.ProviderForLocalConfig(pair.InChild.Addr())
		fullAddr := addrs.AbsProviderConfig{
			Provider: fqn,
			Module:   path,
			Alias:    pair.InChild.Addr().Alias,
		}

		fullParentAddr := addrs.AbsProviderConfig{
			Provider: fqn,
			Module:   parentPath,
			Alias:    pair.InParent.Addr().Alias,
		}

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

		// replace the concrete node with the provider passed in only if it is
		// proxyable
		if concreteProvider != nil {
			if t.proxiable[fullName] {
				g.Replace(concreteProvider, proxy)
				t.providers[fullName] = proxy
			}
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
		mc := t.Config.Descendant(addr.Module)
		if mc == nil {
			log.Printf("[TRACE] ProviderConfigTransformer: no configuration available for %s", addr.String())
			continue
		}

		// Find the localName for the provider fqn
		localName := mc.Module.LocalNameForProvider(addr.Provider)

		// Go through the provider configs to find the matching config
		for _, p := range mc.Module.ProviderConfigs {
			if p.Name == localName && p.Alias == addr.Alias {
				log.Printf("[TRACE] ProviderConfigTransformer: attaching to %q provider configuration from %s", dag.VertexName(v), p.DeclRange)
				apn.AttachProvider(p)
				break
			}
		}
	}

	return nil
}

// externalProviderTransformer adds placeholder graph nodes for any providers
// that were already instantiated and configured by the external caller.
//
// This should typically run before any other transformers that can add
// nodes representing provider configurations, so that the others can notice
// that a node is already present and therefore skip adding a duplicate.
type externalProviderTransformer struct {
	ExternalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
}

func (t *externalProviderTransformer) Transform(g *Graph) error {
	existing := providerVertexMap(g)

	for rootAddr := range t.ExternalProviderConfigs {
		absAddr := rootAddr.AbsProviderConfig()
		if existing, exists := existing[absAddr.String()]; exists {
			// We must not allow a non-external graph node to exist for
			// an externally-configured provider, because that would
			// cause strange things to happen. We shouldn't get here in
			// practice because externalProviderTransformer should be
			// the first transformer that introduces graph nodes representing
			// provider configurations.
			return fmt.Errorf("conflicting %T node for externally-configured provider %s (this is a bug in Terraform)", existing, absAddr)
		}
		abstract := &NodeAbstractProvider{
			Addr: absAddr,
		}
		concrete := &nodeExternalProvider{
			NodeAbstractProvider: abstract,
		}
		g.Add(concrete)
	}
	return nil
}
