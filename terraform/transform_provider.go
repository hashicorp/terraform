package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeProvider is an interface that nodes that can be a provider
// must implement. The ProviderName returned is the name of the provider
// they satisfy.
type GraphNodeProvider interface {
	ProviderName() string
	ProviderConfig() *config.RawConfig
}

// GraphNodeCloseProvider is an interface that nodes that can be a close
// provider must implement. The CloseProviderName returned is the name of
// the provider they satisfy.
type GraphNodeCloseProvider interface {
	CloseProviderName() string
}

// GraphNodeProviderConsumer is an interface that nodes that require
// a provider must implement. ProvidedBy must return the name of the provider
// to use.
type GraphNodeProviderConsumer interface {
	ProvidedBy() []string
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
			for _, p := range pv.ProvidedBy() {
				target := m[providerMapKey(p, pv)]
				if target == nil {
					println(fmt.Sprintf("%#v\n\n%#v", m, providerMapKey(p, pv)))
					err = multierror.Append(err, fmt.Errorf(
						"%s: provider %s couldn't be found",
						dag.VertexName(v), p))
					continue
				}

				g.Connect(dag.BasicEdge(v, target))
			}
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
	cpm := closeProviderVertexMap(g)
	var err error
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProviderConsumer); ok {
			for _, p := range pv.ProvidedBy() {
				key := p
				source := cpm[key]

				if source == nil {
					// Create a new graphNodeCloseProvider and add it to the graph
					source = &graphNodeCloseProvider{ProviderNameValue: p}
					g.Add(source)

					// Close node needs to depend on provider
					provider, ok := pm[key]
					if !ok {
						err = multierror.Append(err, fmt.Errorf(
							"%s: provider %s couldn't be found for closing",
							dag.VertexName(v), p))
						continue
					}
					g.Connect(dag.BasicEdge(source, provider))

					// Make sure we also add the new graphNodeCloseProvider to the map
					// so we don't create and add any duplicate graphNodeCloseProviders.
					cpm[key] = source
				}

				// Close node depends on all nodes provided by the provider
				g.Connect(dag.BasicEdge(source, v))
			}
		}
	}

	return err
}

// MissingProviderTransformer is a GraphTransformer that adds nodes
// for missing providers into the graph. Specifically, it creates provider
// configuration nodes for all the providers that we support. These are
// pruned later during an optimization pass.
type MissingProviderTransformer struct {
	// Providers is the list of providers we support.
	Providers []string

	// AllowAny will not check that a provider is supported before adding
	// it to the graph.
	AllowAny bool

	// Concrete, if set, overrides how the providers are made.
	Concrete ConcreteProviderNodeFunc
}

func (t *MissingProviderTransformer) Transform(g *Graph) error {
	// Initialize factory
	if t.Concrete == nil {
		t.Concrete = func(a *NodeAbstractProvider) dag.Vertex {
			return &graphNodeProvider{ProviderNameValue: a.NameValue}
		}
	}

	// Create a set of our supported providers
	supported := make(map[string]struct{}, len(t.Providers))
	for _, v := range t.Providers {
		supported[v] = struct{}{}
	}

	// Get the map of providers we already have in our graph
	m := providerVertexMap(g)

	// Go through all the provider consumers and make sure we add
	// that provider if it is missing. We use a for loop here instead
	// of "range" since we'll modify check as we go to add more to check.
	check := g.Vertices()
	for i := 0; i < len(check); i++ {
		v := check[i]

		pv, ok := v.(GraphNodeProviderConsumer)
		if !ok {
			continue
		}

		// If this node has a subpath, then we use that as a prefix
		// into our map to check for an existing provider.
		var path []string
		if sp, ok := pv.(GraphNodeSubPath); ok {
			raw := normalizeModulePath(sp.Path())
			if len(raw) > len(rootModulePath) {
				path = raw
			}
		}

		for _, p := range pv.ProvidedBy() {
			key := providerMapKey(p, pv)
			if _, ok := m[key]; ok {
				// This provider already exists as a configure node
				continue
			}

			// If the provider has an alias in it, we just want the type
			ptype := p
			if idx := strings.IndexRune(p, '.'); idx != -1 {
				ptype = p[:idx]
			}

			if !t.AllowAny {
				if _, ok := supported[ptype]; !ok {
					// If we don't support the provider type, skip it.
					// Validation later will catch this as an error.
					continue
				}
			}

			// Add the missing provider node to the graph
			v := t.Concrete(&NodeAbstractProvider{
				NameValue: p,
				PathValue: path,
			}).(dag.Vertex)
			if len(path) > 0 {
				if fn, ok := v.(GraphNodeFlattenable); ok {
					var err error
					v, err = fn.Flatten(path)
					if err != nil {
						return err
					}
				}

				// We'll need the parent provider as well, so let's
				// add a dummy node to check to make sure that we add
				// that parent provider.
				check = append(check, &graphNodeProviderConsumerDummy{
					ProviderValue: p,
					PathValue:     path[:len(path)-1],
				})
			}

			m[key] = g.Add(v)
		}
	}

	return nil
}

// ParentProviderTransformer connects provider nodes to their parents.
//
// This works by finding nodes that are both GraphNodeProviders and
// GraphNodeSubPath. It then connects the providers to their parent
// path.
type ParentProviderTransformer struct{}

func (t *ParentProviderTransformer) Transform(g *Graph) error {
	// Make a mapping of path to dag.Vertex, where path is: "path.name"
	m := make(map[string]dag.Vertex)

	// Also create a map that maps a provider to its parent
	parentMap := make(map[dag.Vertex]string)
	for _, raw := range g.Vertices() {
		// If it is the flat version, then make it the non-flat version.
		// We eventually want to get rid of the flat version entirely so
		// this is a stop-gap while it still exists.
		var v dag.Vertex = raw
		if f, ok := v.(*graphNodeProviderFlat); ok {
			v = f.graphNodeProvider
		}

		// Only care about providers
		pn, ok := v.(GraphNodeProvider)
		if !ok || pn.ProviderName() == "" {
			continue
		}

		// Also require a subpath, if there is no subpath then we
		// just totally ignore it. The expectation of this transform is
		// that it is used with a graph builder that is already flattened.
		var path []string
		if pn, ok := raw.(GraphNodeSubPath); ok {
			path = pn.Path()
		}
		path = normalizeModulePath(path)

		// Build the key with path.name i.e. "child.subchild.aws"
		key := fmt.Sprintf("%s.%s", strings.Join(path, "."), pn.ProviderName())
		m[key] = raw

		// Determine the parent if we're non-root. This is length 1 since
		// the 0 index should be "root" since we normalize above.
		if len(path) > 1 {
			path = path[:len(path)-1]
			key := fmt.Sprintf("%s.%s", strings.Join(path, "."), pn.ProviderName())
			parentMap[raw] = key
		}
	}

	// Connect!
	for v, key := range parentMap {
		if parent, ok := m[key]; ok {
			g.Connect(dag.BasicEdge(v, parent))
		}
	}

	return nil
}

// PruneProviderTransformer is a GraphTransformer that prunes all the
// providers that aren't needed from the graph. A provider is unneeded if
// no resource or module is using that provider.
type PruneProviderTransformer struct{}

func (t *PruneProviderTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about the providers
		if pn, ok := v.(GraphNodeProvider); !ok || pn.ProviderName() == "" {
			continue
		}
		// Does anything depend on this? If not, then prune it.
		if s := g.UpEdges(v); s.Len() == 0 {
			if nv, ok := v.(dag.NamedVertex); ok {
				log.Printf("[DEBUG] Pruning provider with no dependencies: %s", nv.Name())
			}
			g.Remove(v)
		}
	}

	return nil
}

// providerMapKey is a helper that gives us the key to use for the
// maps returned by things such as providerVertexMap.
func providerMapKey(k string, v dag.Vertex) string {
	pathPrefix := ""
	if sp, ok := v.(GraphNodeSubPath); ok {
		raw := normalizeModulePath(sp.Path())
		if len(raw) > len(rootModulePath) {
			pathPrefix = modulePrefixStr(raw) + "."
		}
	}

	return pathPrefix + k
}

func providerVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvider); ok {
			key := pv.ProviderName()

			// This special case is because the new world view of providers
			// is that they should return only their pure name (not the full
			// module path with ProviderName). Working towards this future.
			if _, ok := v.(*NodeApplyableProvider); ok {
				key = providerMapKey(pv.ProviderName(), v)
			}

			m[key] = v
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

type graphNodeProvider struct {
	ProviderNameValue string
}

func (n *graphNodeProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.ProviderNameValue)
}

// GraphNodeEvalable impl.
func (n *graphNodeProvider) EvalTree() EvalNode {
	return ProviderEvalTree(n.ProviderNameValue, nil)
}

// GraphNodeDependable impl.
func (n *graphNodeProvider) DependableName() []string {
	return []string{n.Name()}
}

// GraphNodeProvider
func (n *graphNodeProvider) ProviderName() string {
	return n.ProviderNameValue
}

func (n *graphNodeProvider) ProviderConfig() *config.RawConfig {
	return nil
}

// GraphNodeDotter impl.
func (n *graphNodeProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}

// GraphNodeDotterOrigin impl.
func (n *graphNodeProvider) DotOrigin() bool {
	return true
}

// GraphNodeFlattenable impl.
func (n *graphNodeProvider) Flatten(p []string) (dag.Vertex, error) {
	return &graphNodeProviderFlat{
		graphNodeProvider: n,
		PathValue:         p,
	}, nil
}

// Same as graphNodeMissingProvider, but for flattening
type graphNodeProviderFlat struct {
	*graphNodeProvider

	PathValue []string
}

func (n *graphNodeProviderFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.graphNodeProvider.Name())
}

func (n *graphNodeProviderFlat) Path() []string {
	return n.PathValue
}

func (n *graphNodeProviderFlat) ProviderName() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue),
		n.graphNodeProvider.ProviderName())
}

// GraphNodeDependable impl.
func (n *graphNodeProviderFlat) DependableName() []string {
	return []string{n.Name()}
}

func (n *graphNodeProviderFlat) DependentOn() []string {
	var result []string

	// If we're in a module, then depend on all parent providers. Some of
	// these may not exist, hence we depend on all of them.
	for i := len(n.PathValue); i > 1; i-- {
		prefix := modulePrefixStr(n.PathValue[:i-1])
		result = modulePrefixList(n.graphNodeProvider.DependableName(), prefix)
	}

	return result
}

// graphNodeProviderConsumerDummy is a struct that never enters the real
// graph (though it could to no ill effect). It implements
// GraphNodeProviderConsumer and GraphNodeSubpath as a way to force
// certain transformations.
type graphNodeProviderConsumerDummy struct {
	ProviderValue string
	PathValue     []string
}

func (n *graphNodeProviderConsumerDummy) Path() []string {
	return n.PathValue
}

func (n *graphNodeProviderConsumerDummy) ProvidedBy() []string {
	return []string{n.ProviderValue}
}
