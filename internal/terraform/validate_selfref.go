// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// validateSelfRef checks to ensure that expressions within a particular
// referencable block do not reference that same block.
func validateSelfRef(addr addrs.Referenceable, config hcl.Body, providerSchema providers.ProviderSchema) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	addrStrs := make([]string, 0, 1)
	addrStrs = append(addrStrs, addr.String())
	switch tAddr := addr.(type) {
	case addrs.ResourceInstance:
		// A resource instance may not refer to its containing resource either.
		addrStrs = append(addrStrs, tAddr.ContainingResource().String())
	}

	var schema *configschema.Block
	switch tAddr := addr.(type) {
	case addrs.Resource:
		schema, _ = providerSchema.SchemaForResourceAddr(tAddr)
	case addrs.ResourceInstance:
		schema, _ = providerSchema.SchemaForResourceAddr(tAddr.ContainingResource())
	}

	if schema == nil {
		diags = diags.Append(fmt.Errorf("no schema available for %s to validate for self-references; this is a bug in Terraform and should be reported", addr))
		return diags
	}

	refs, _ := lang.ReferencesInBlock(addrs.ParseRef, config, schema)
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

	return diags
}

// Legacy provisioner configurations may refer to single instances using the
// resource address. We need to filter these out from the reported references
// to prevent cycles.
func filterSelfRefs(self addrs.Resource, refs []*addrs.Reference) []*addrs.Reference {
	for i := 0; i < len(refs); i++ {
		ref := refs[i]

		var subject addrs.Resource
		switch subj := ref.Subject.(type) {
		case addrs.Resource:
			subject = subj
		case addrs.ResourceInstance:
			subject = subj.ContainingResource()
		default:
			continue
		}

		if self.Equal(subject) {
			tail := len(refs) - 1

			refs[i], refs[tail] = refs[tail], refs[i]
			refs = refs[:tail]
		}
	}
	return refs
}
