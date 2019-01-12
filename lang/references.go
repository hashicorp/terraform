package lang

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

// References finds all of the references in the given set of traversals,
// returning diagnostics if any of the traversals cannot be interpreted as a
// reference.
//
// This function does not do any de-duplication of references, since references
// have source location information embedded in them and so any invalid
// references that are duplicated should have errors reported for each
// occurence.
//
// If the returned diagnostics contains errors then the result may be
// incomplete or invalid. Otherwise, the returned slice has one reference per
// given traversal, though it is not guaranteed that the references will
// appear in the same order as the given traversals.
func References(traversals []hcl.Traversal) ([]*addrs.Reference, tfdiags.Diagnostics) {
	if len(traversals) == 0 {
		return nil, nil
	}

	var diags tfdiags.Diagnostics
	refs := make([]*addrs.Reference, 0, len(traversals))

	for _, traversal := range traversals {
		ref, refDiags := addrs.ParseRef(traversal)
		diags = diags.Append(refDiags)
		if ref == nil {
			continue
		}
		refs = append(refs, ref)
	}

	return refs, diags
}

// ReferencesInBlock is a helper wrapper around References that first searches
// the given body for traversals, before converting those traversals to
// references.
//
// A block schema must be provided so that this function can determine where in
// the body variables are expected.
func ReferencesInBlock(body hcl.Body, schema *configschema.Block) ([]*addrs.Reference, tfdiags.Diagnostics) {
	if body == nil {
		return nil, nil
	}
	spec := schema.DecoderSpec()
	traversals := hcldec.Variables(body, spec)
	return References(traversals)
}

// ReferencesInExpr is a helper wrapper around References that first searches
// the given expression for traversals, before converting those traversals
// to references.
func ReferencesInExpr(expr hcl.Expression) ([]*addrs.Reference, tfdiags.Diagnostics) {
	if expr == nil {
		return nil, nil
	}
	traversals := expr.Variables()
	return References(traversals)
}
