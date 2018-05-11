package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalValidateSelfRef is an EvalNode implementation that checks to ensure that
// expressions within a particular referencable block do not reference that
// same block.
type EvalValidateSelfRef struct {
	Addr           addrs.Referenceable
	Config         hcl.Body
	ProviderSchema **ProviderSchema
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

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema unavailable while validating %s for self-references; this is a bug in Terraform and should be reported", addr)
	}

	providerSchema := *n.ProviderSchema
	var schema *configschema.Block
	switch tAddr := addr.(type) {
	case addrs.Resource:
		switch tAddr.Mode {
		case addrs.ManagedResourceMode:
			schema = providerSchema.ResourceTypes[tAddr.Type]
		case addrs.DataResourceMode:
			schema = providerSchema.DataSources[tAddr.Type]
		}
	case addrs.ResourceInstance:
		switch tAddr.Resource.Mode {
		case addrs.ManagedResourceMode:
			schema = providerSchema.ResourceTypes[tAddr.Resource.Type]
		case addrs.DataResourceMode:
			schema = providerSchema.DataSources[tAddr.Resource.Type]
		}
	}

	if schema == nil {
		return nil, fmt.Errorf("no schema available for %s to validate for self-references; this is a bug in Terraform and should be reported", addr)
	}

	refs, _ := lang.ReferencesInBlock(n.Config, schema)
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
