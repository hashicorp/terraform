package terraform

import (
	"fmt"
	"strings"
)

// ImportProviderValidateTransformer is a GraphTransformer that goes through
// the providers in the graph and validates that they only depend on variables.
type ImportProviderValidateTransformer struct{}

func (t *ImportProviderValidateTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about providers
		pv, ok := v.(GraphNodeProvider)
		if !ok {
			continue
		}

		// We only care about providers that reference things
		rn, ok := pv.(GraphNodeReferencer)
		if !ok {
			continue
		}

		for _, ref := range rn.References() {
			if !strings.HasPrefix(ref, "var.") {
				return fmt.Errorf(
					"Provider %q depends on non-var %q. Providers for import can currently\n"+
						"only depend on variables or must be hardcoded. You can stop import\n"+
						"from loading configurations by specifying `-config=\"\"`.",
					pv.ProviderName(), ref)
			}
		}
	}

	return nil
}
