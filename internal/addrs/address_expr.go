// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// AddressExpr is an expression referring to an address, in which each
// instance key can be given by an arbitrary expression rather than only by a
// constant, such as aws_instance.example[each.key].
//
// An AddressExpr has the same steps as the traversal for the address, except
// that each instance key given by an expression is kept as that expression
// for the caller to evaluate.
type AddressExpr []AddressExprStep

// AddressExprStep is a single step of an AddressExpr, which is either a
// static traversal step or an instance key given by an expression.
type AddressExprStep struct {
	// Step is the static traversal step, when Key is nil.
	Step hcl.Traverser

	// Key is the expression for an instance key which isn't a constant.
	Key hcl.Expression
}

// ParseAddressExpr returns the AddressExpr for the given expression.
//
// Expressions which aren't from the HCL native syntax must be static
// traversals, without any instance keys given by expressions.
func ParseAddressExpr(expr hcl.Expression) (AddressExpr, hcl.Diagnostics) {
	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		ret, diags := ParseAddressExpr(e.Source)
		if diags.HasErrors() {
			return nil, diags
		}
		return ret.appendTraversal(e.Traversal), diags

	case *hclsyntax.ScopeTraversalExpr:
		return AddressExpr(nil).appendTraversal(e.Traversal), nil

	case *hclsyntax.IndexExpr:
		ret, diags := ParseAddressExpr(e.Collection)
		if diags.HasErrors() {
			return nil, diags
		}
		return append(ret, AddressExprStep{Key: e.Key}), diags

	default:
		traversal, diags := hcl.AbsTraversalForExpr(expr)
		if diags.HasErrors() {
			return nil, diags
		}
		return AddressExpr(nil).appendTraversal(traversal), diags
	}
}

func (e AddressExpr) appendTraversal(traversal hcl.Traversal) AddressExpr {
	for _, step := range traversal {
		e = append(e, AddressExprStep{Step: step})
	}
	return e
}

// Traversal returns the traversal for the address, where each instance key
// given by an expression is replaced with the value returned by evalKey. If
// that value is unknown, the key is a wildcard when the traversal is parsed
// as a pattern or partial-expanded address.
//
// If evalKey is nil, each instance key given by an expression is instead a
// wildcard, as if written as [*].
func (e AddressExpr) Traversal(evalKey func(hcl.Expression) (cty.Value, hcl.Diagnostics)) (hcl.Traversal, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := make(hcl.Traversal, 0, len(e))
	for _, step := range e {
		if step.Key == nil {
			ret = append(ret, step.Step)
			continue
		}

		rng := step.Key.Range()
		if evalKey == nil {
			ret = append(ret, hcl.TraverseSplat{SrcRange: rng})
			continue
		}

		key, moreDiags := evalKey(step.Key)
		diags = append(diags, moreDiags...)
		if moreDiags.HasErrors() {
			return nil, diags
		}
		ret = append(ret, hcl.TraverseIndex{Key: key, SrcRange: rng})
	}
	return ret, diags
}
