package lang

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang/blocktoattr"
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

	// We use blocktoattr.ExpandedVariables instead of hcldec.Variables or
	// dynblock.VariablesHCLDec here because when we evaluate a block we'll
	// first apply the dynamic block extension and _then_ the blocktoattr
	// transform, and so blocktoattr.ExpandedVariables takes into account
	// both of those transforms when it analyzes the body to ensure we find
	// all of the references as if they'd already moved into their final
	// locations, even though we can't expand dynamic blocks yet until we
	// already know which variables are required.
	//
	// The set of cases we want to detect here is covered by the tests for
	// the plan graph builder in the main 'terraform' package, since it's
	// in a better position to test this due to having mock providers etc
	// available.
	traversals := blocktoattr.ExpandedVariables(body, schema)
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
