// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
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

	refs, _ := langrefs.ReferencesInBlock(addrs.ParseRef, config, schema)
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

// validateMetaSelfRef checks to ensure that a specific meta expression (count /
// for_each) does not reference the resource it is attached to. The behaviour
// is slightly different from validateSelfRef in that this function is only ever
// called from static contexts (ie. before expansion) and as such the address is
// always a Resource.
//
// This also means that often the references will be to instances of the
// resource, so we need to unpack these to the containing resource to compare
// against the static resource. From the perspective of this function
// `test_resource.foo[4]` is considered to be a self reference to
// `test_resource.foo`, in which is a significant behaviour change to
// validateSelfRef.
func validateMetaSelfRef(addr addrs.Resource, expr hcl.Expression) tfdiags.Diagnostics {
	return validateSelfRefFromExprInner(addr, expr, func(ref *addrs.Reference) *hcl.Diagnostic {
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Self-referential block",
			Detail:   fmt.Sprintf("Configuration for %s may not refer to itself.", addr.String()),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		}
	})
}

// validateImportSelfRef is similar to validateMetaSelfRef except it
// tweaks the error message slightly to reflect the self-reference is coming
// from an import block instead of directly from the resource. All the same
// caveats apply as validateMetaSelfRef.
func validateImportSelfRef(addr addrs.Resource, expr hcl.Expression) tfdiags.Diagnostics {
	return validateSelfRefFromExprInner(addr, expr, func(ref *addrs.Reference) *hcl.Diagnostic {
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID cannot reference the resource being imported.",
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		}
	})
}

// validateSelfRefFromExprInner is a helper function that takes an address and
// an expression and returns diagnostics for self-references in the expression.
//
// This should only be called via validateMetaSelfRef and validateImportSelfRef,
// do not access this function directly.
func validateSelfRefFromExprInner(addr addrs.Resource, expr hcl.Expression, diag func(ref *addrs.Reference) *hcl.Diagnostic) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	for _, ref := range refs {
		var target addrs.Resource
		switch t := ref.Subject.(type) {
		case addrs.ResourceInstance:
			// Automatically unpack an instance reference to its containing
			// resource, since we're only comparing against the static resource.
			target = t.Resource
		case addrs.Resource:
			target = t
		default:
			// Anything else cannot be a self-reference.
			continue
		}

		if target.Equal(addr) {
			diags = diags.Append(diag(ref))
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
