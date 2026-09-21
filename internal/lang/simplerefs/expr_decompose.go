// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package simplerefs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// DecomposeTraversals analyzes a non-simple config expression and returns the
// set of absolute traversals it could evaluate to, restricted to the
// "reference-preserving" subset of expressions whose provenance can be decided
// statically.
//
// It returns:
//   - traversals: every resource-attribute traversal the expression can yield
//     (index steps stripped, so comparison is config-address granular).
//   - hasNonRef: true if the expression has at least one branch that is a
//     constant/literal (i.e. a value that is definitively not a reference).
//   - ok: true if the expression is drawn entirely from the reference-preserving
//     grammar and could therefore be analyzed. When ok is false the expression
//     is opaque (a non-preserving function, an interpolated template, etc.) and
//     callers must treat it as unknown.
//
// The supported grammar is: resource traversals (with index steps), splats,
// conditionals (union of both branches), and a curated allowlist of
// reference-preserving functions. Anything else yields ok=false. This keeps
// provenance sound: a caller only treats the result as a definite reference when
// the traversals resolve to a single homogeneous resource with no non-reference
// branch; every ambiguous shape falls back to unknown.
func DecomposeTraversals(expr hcl.Expression) (traversals []hcl.Traversal, hasNonRef bool, ok bool) {
	return decompose(expr, nil, false)
}

// DecomposeMembershipTraversals is the membership (expansion-connector) variant.
// A membership connector (`any_of`/`has`) matches when the target is among the
// *coexisting* element references of a collection expression. Unlike the scalar
// form, having many distinct references is expected (each element is a real
// member), so the caller compares set membership rather than requiring a single
// reference. To stay sound it rejects *choice* constructs — conditionals and the
// selection functions (`element`, `one`, `try`, `coalesce`, `slice`, …) — whose
// result is one of several alternatives not known until apply; those yield
// ok=false so the caller defers/halts instead of over-claiming membership.
func DecomposeMembershipTraversals(expr hcl.Expression) (traversals []hcl.Traversal, hasNonRef bool, ok bool) {
	return decompose(expr, nil, true)
}

// decompose is the env-threaded implementation. env maps `for`-expression
// iterator variable names to the base traversals their elements are drawn from,
// so a reference to the iterator inside the `for` body (e.g. `s.id`) resolves to
// the collection's resource attribute. When membership is true, choice
// constructs (conditionals, selection functions) are rejected so that only
// genuinely coexisting element references are collected.
func decompose(expr hcl.Expression, env map[string][]hcl.Traversal, membership bool) (traversals []hcl.Traversal, hasNonRef bool, ok bool) {
	switch e := hcl.UnwrapExpression(expr).(type) {
	case *hclsyntax.ParenthesesExpr:
		return decompose(e.Expression, env, membership)

	case *hclsyntax.ScopeTraversalExpr:
		tr := e.Traversal
		// If the traversal is rooted at a `for` iterator variable, substitute
		// the collection's base traversals and append the remaining attributes.
		if len(tr) > 0 {
			if root, isRoot := tr[0].(hcl.TraverseRoot); isRoot {
				if bases, isVar := env[root.Name]; isVar {
					return appendRel(bases, StripIndexSteps(tr[1:])), false, true
				}
			}
		}
		return []hcl.Traversal{StripIndexSteps(tr)}, false, true

	case *hclsyntax.RelativeTraversalExpr:
		base, nn, bok := decompose(e.Source, env, membership)
		if !bok {
			return nil, false, false
		}
		return appendRel(base, StripIndexSteps(e.Traversal)), nn, true

	case *hclsyntax.IndexExpr:
		// The index key selects an instance; drop it and keep the collection's
		// reference (config-address granular).
		return decompose(e.Collection, env, membership)

	case *hclsyntax.SplatExpr:
		base, nn, bok := decompose(e.Source, env, membership)
		if !bok {
			return nil, false, false
		}
		rel, rok := anonRelativeTraversal(e.Each)
		if !rok {
			return nil, false, false
		}
		return appendRel(base, rel), nn, true

	case *hclsyntax.TupleConsExpr:
		// A tuple/list literal ([a, b, …]) is reference-preserving: its value is
		// the collection of its elements, so union their decompositions.
		var all []hcl.Traversal
		nn := false
		for _, el := range e.Exprs {
			et, en, eok := decompose(el, env, membership)
			if !eok {
				return nil, false, false
			}
			all = append(all, et...)
			nn = nn || en
		}
		return all, nn, true

	case *hclsyntax.ObjectConsExpr:
		// An object/map literal ({ k = v, … }). Its provenance is the union of
		// its item VALUES; structural keys are ignored (they are not the value a
		// connector compares against).
		var all []hcl.Traversal
		nn := false
		for _, item := range e.Items {
			vt, vn, vok := decompose(item.ValueExpr, env, membership)
			if !vok {
				return nil, false, false
			}
			all = append(all, vt...)
			nn = nn || vn
		}
		return all, nn, true

	case *hclsyntax.ConditionalExpr:
		if membership {
			// A conditional chooses one branch; its members are not all present,
			// so membership cannot be decided statically.
			return nil, false, false
		}
		tt, tn, tok := decompose(e.TrueResult, env, membership)
		ft, fn, fok := decompose(e.FalseResult, env, membership)
		if !tok || !fok {
			return nil, false, false
		}
		return append(tt, ft...), tn || fn, true

	case *hclsyntax.ForExpr:
		// [for v in COLL : VAL] / {for k, v in COLL : KEY => VAL}. The iterator
		// variable v stands for an element of COLL, so bind it to COLL's base
		// resource(s) and decompose the produced key/value in that scope.
		collBase, collNonRef, collOk := decompose(e.CollExpr, env, membership)
		if !collOk {
			return nil, false, false
		}
		child := make(map[string][]hcl.Traversal, len(env)+1)
		for k, v := range env {
			child[k] = v
		}
		if e.ValVar != "" {
			child[e.ValVar] = collBase
		}
		// The key variable is a collection index/key, not a resource; leaving it
		// out of the env makes any reference to it resolve to a non-resource
		// (unknown), which is sound.

		vt, vn, vok := decompose(e.ValExpr, child, membership)
		if !vok {
			return nil, false, false
		}
		all := vt
		nn := collNonRef || vn
		if e.KeyExpr != nil {
			kt, kn, kok := decompose(e.KeyExpr, child, membership)
			if !kok {
				return nil, false, false
			}
			all = append(all, kt...)
			nn = nn || kn
		}
		// The `if` condition only filters elements; it does not contribute to
		// the produced value's provenance, so it is ignored.
		return all, nn, true

	case *hclsyntax.FunctionCallExpr:
		sel, known := refPreservingFuncs[e.Name]
		if !known {
			return nil, false, false
		}
		if membership {
			// In membership mode only *set-preserving* functions are safe: they
			// keep every element (possibly reordered/retyped/deduplicated) so
			// the target is a member iff it references one of the elements.
			// Selection/choice functions (element, one, slice, chunklist, try,
			// coalesce, coalescelist) return a subset/one-of not known until
			// apply, so they are rejected.
			if _, ok := membershipPreservingFuncs[e.Name]; !ok {
				return nil, false, false
			}
		}
		var all []hcl.Traversal
		nn := false
		for i, arg := range e.Args {
			if sel == firstArgOnly && i != 0 {
				// Control args (index, start/end, chunk size) do not contribute
				// values, so ignore them entirely.
				continue
			}
			at, an, aok := decompose(arg, env, membership)
			if !aok {
				return nil, false, false
			}
			all = append(all, at...)
			nn = nn || an
		}
		return all, nn, true

	case *hclsyntax.LiteralValueExpr:
		return nil, true, true

	case *hclsyntax.TemplateExpr:
		if e.IsStringLiteral() {
			return nil, true, true
		}
		// An interpolated template builds a derived string; treat as opaque.
		return nil, false, false

	case *hclsyntax.TemplateWrapExpr:
		// A bare "${x}" wrapper is just its inner expression.
		return decompose(e.Wrapped, env, membership)

	default:
		return nil, false, false
	}
}

// appendRel appends the relative traversal rel to a copy of every base
// traversal.
func appendRel(base []hcl.Traversal, rel hcl.Traversal) []hcl.Traversal {
	if len(rel) == 0 {
		return base
	}
	out := make([]hcl.Traversal, 0, len(base))
	for _, b := range base {
		combined := make(hcl.Traversal, 0, len(b)+len(rel))
		combined = append(combined, b...)
		combined = append(combined, rel...)
		out = append(out, combined)
	}
	return out
}

// anonRelativeTraversal extracts the per-element attribute traversal from a
// splat's Each expression, which is expressed relative to an anonymous symbol
// (e.g. the `.id` in `aws_subnet.this[*].id`). It supports only a direct
// anon-symbol attribute chain; nested splats or other shapes return false.
func anonRelativeTraversal(each hcl.Expression) (hcl.Traversal, bool) {
	switch e := hcl.UnwrapExpression(each).(type) {
	case *hclsyntax.AnonSymbolExpr:
		return hcl.Traversal{}, true
	case *hclsyntax.RelativeTraversalExpr:
		if _, ok := hcl.UnwrapExpression(e.Source).(*hclsyntax.AnonSymbolExpr); ok {
			return StripIndexSteps(e.Traversal), true
		}
		return nil, false
	default:
		return nil, false
	}
}

// StripIndexSteps returns the traversal with any index steps removed, leaving
// only the root and attribute steps so the reference reduces to its config
// resource.
func StripIndexSteps(t hcl.Traversal) hcl.Traversal {
	out := make(hcl.Traversal, 0, len(t))
	for _, step := range t {
		if _, ok := step.(hcl.TraverseIndex); ok {
			continue
		}
		out = append(out, step)
	}
	return out
}

type argSelection int

const (
	allArgs argSelection = iota
	firstArgOnly
)

// refPreservingFuncs is the curated allowlist of functions whose result's
// provenance is the union of the provenance of their (value-contributing)
// arguments. firstArgOnly functions take a single collection plus control args
// (index/size) that do not contribute references; allArgs functions combine
// every argument. Functions not listed here are opaque and force an unknown.
var refPreservingFuncs = map[string]argSelection{
	"element":      firstArgOnly,
	"one":          firstArgOnly,
	"flatten":      firstArgOnly,
	"tolist":       firstArgOnly,
	"toset":        firstArgOnly,
	"slice":        firstArgOnly,
	"sort":         firstArgOnly,
	"reverse":      firstArgOnly,
	"distinct":     firstArgOnly,
	"compact":      firstArgOnly,
	"chunklist":    firstArgOnly,
	"try":          allArgs,
	"coalesce":     allArgs,
	"coalescelist": allArgs,
	"concat":       allArgs,
}

// membershipPreservingFuncs is the subset of refPreservingFuncs that keep the
// *entire* element set (only reordering, retyping, flattening, or removing
// duplicates/nulls), so a target is a member of the result iff it references one
// of the input elements. Functions that select or drop elements by an
// apply-time value (element, one, slice, chunklist) or choose among alternatives
// (try, coalesce, coalescelist) are intentionally excluded: they are unsound for
// membership because we cannot know statically which elements survive.
var membershipPreservingFuncs = map[string]struct{}{
	"concat":   {},
	"flatten":  {},
	"tolist":   {},
	"toset":    {},
	"distinct": {},
	"compact":  {},
	"sort":     {},
	"reverse":  {},
}

// IsMembershipPreservingFunc reports whether name is a function that preserves
// the full element set of its collection argument (so a target is a member of
// the result iff it references one of the inputs). Callers use it to recognize a
// membership (expansion) connector expression.
func IsMembershipPreservingFunc(name string) bool {
	_, ok := membershipPreservingFuncs[name]
	return ok
}
