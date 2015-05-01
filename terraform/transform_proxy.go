package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeProxy must be implemented by nodes that are proxies.
//
// A node that is a proxy says that anything that depends on this
// node (the proxy), should also copy all the things that the proxy
// itself depends on. Example:
//
//    A => proxy => C
//
// Should transform into (two edges):
//
//    A => proxy => C
//    A => C
//
// The purpose for this is because some transforms only look at direct
// edge connections and the proxy generally isn't meaningful in those
// situations, so we should complete all the edges.
type GraphNodeProxy interface {
	Proxy() bool
}

// ProxyTransformer is a transformer that goes through the graph, finds
// vertices that are marked as proxies, and connects through their
// dependents. See above for what a proxy is.
type ProxyTransformer struct{}

func (t *ProxyTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		pn, ok := v.(GraphNodeProxy)
		if !ok {
			continue
		}

		// If we don't want to be proxies, don't do it
		if !pn.Proxy() {
			continue
		}

		// Connect all the things that depend on this to things that
		// we depend on as the proxy. See docs for GraphNodeProxy for
		// a visual explanation.
		for _, s := range g.UpEdges(v).List() {
			for _, t := range g.DownEdges(v).List() {
				g.Connect(GraphProxyEdge{
					Edge: dag.BasicEdge(s, t),
				})
			}
		}
	}

	return nil
}

// GraphProxyEdge is the edge that is used for proxied edges.
type GraphProxyEdge struct {
	dag.Edge
}
