package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/lang"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
)

// EvalValidateSelfRef is an EvalNode implementation that checks to ensure that
// expressions within a particular referencable block do not reference that
// same block.
type EvalValidateSelfRef struct {
	Addr   addrs.Referenceable
	Config hcl.Body
	Schema *configschema.Block
}

func (n *EvalValidateSelfRef) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	addr := n.Addr

	addrStrs := make([]string, 0, 1)
	addrStrs = append(addrStrs, addr.String())
	switch tAddr := addr.(type) {
	case addrs.ResourceInstance:
		// A resource instance may not refer to its containing resource either.
		addrStrs = append(addrStrs, tAddr.ContainingResource().String())
	}

	refs, _ := lang.ReferencesInBlock(n.Config, n.Schema)
	for _, ref := range refs {
		for _, addrStr := range addrStrs {
			if ref.Subject.String() == addrStr {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Self-referential block",
					Detail:   fmt.Sprintf("Configuration for %s may not refer to itself.", addrStr),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
			}
		}
	}

	return nil, diags.NonFatalErr()
}
