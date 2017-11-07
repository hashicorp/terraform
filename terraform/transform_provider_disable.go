package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// DisableProviderTransformer "disables" any providers that are not actually
// used by anything, and provider proxies. This avoids the provider being
// initialized and configured.  This both saves resources but also avoids
// errors since configuration may imply initialization which may require auth.
type DisableProviderTransformer struct{}

func (t *DisableProviderTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about providers
		pn, ok := v.(GraphNodeProvider)
		if !ok || pn.ProviderName() == "" {
			continue
		}

		// remove the proxy nodes now that we're done with them
		if pn, ok := v.(*graphNodeProxyProvider); ok {
			g.Remove(pn)
			continue
		}

		// If we have dependencies, then don't disable
		if g.UpEdges(v).Len() > 0 {
			continue
		}

		// Get the path
		var path []string
		if pn, ok := v.(GraphNodeSubPath); ok {
			path = pn.Path()
		}

		// Disable the provider by replacing it with a "disabled" provider
		disabled := &NodeDisabledProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				NameValue: pn.ProviderName(),
				PathValue: path,
			},
		}

		if !g.Replace(v, disabled) {
			panic(fmt.Sprintf(
				"vertex disappeared from under us: %s",
				dag.VertexName(v)))
		}
	}

	return nil
}
