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

	addrStr := addr.String()

	refs, _ := lang.ReferencesInBlock(n.Config, n.Schema)
	for _, ref := range refs {
		if ref.Subject.String() == addrStr {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self-referential block",
				Detail:   fmt.Sprintf("Configuration for %s may not refer to itself.", addrStr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return nil, diags.NonFatalErr()
}
