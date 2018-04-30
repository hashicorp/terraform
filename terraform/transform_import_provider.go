package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// ImportProviderValidateTransformer is a GraphTransformer that goes through
// the providers in the graph and validates that they only depend on variables.
type ImportProviderValidateTransformer struct{}

func (t *ImportProviderValidateTransformer) Transform(g *Graph) error {
	var diags tfdiags.Diagnostics

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
			if _, ok := ref.Subject.(addrs.InputVariable); !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider dependency for import",
					Detail:   fmt.Sprintf("The configuration for %s depends on %s. Providers used with import must either have literal configuration or refer only to input variables.", pv.ProviderAddr(), ref.Subject.String()),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
			}
		}
	}

	return diags.Err()
}
