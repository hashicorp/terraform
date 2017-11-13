package hcldec

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// Decode interprets the given body using the given specification and returns
// the resulting value. If the given body is not valid per the spec, error
// diagnostics are returned and the returned value is likely to be incomplete.
//
// The ctx argument may be nil, in which case any references to variables or
// functions will produce error diagnostics.
func Decode(body hcl.Body, spec Spec, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	val, _, diags := decode(body, nil, ctx, spec, false)
	return val, diags
}

// PartialDecode is like Decode except that it permits "leftover" items in
// the top-level body, which are returned as a new body to allow for
// further processing.
//
// Any descendent block bodies are _not_ decoded partially and thus must
// be fully described by the given specification.
func PartialDecode(body hcl.Body, spec Spec, ctx *hcl.EvalContext) (cty.Value, hcl.Body, hcl.Diagnostics) {
	return decode(body, nil, ctx, spec, true)
}

// ImpliedType returns the value type that should result from decoding the
// given spec.
func ImpliedType(spec Spec) cty.Type {
	return impliedType(spec)
}

// SourceRange interprets the given body using the given specification and
// then returns the source range of the value that would be used to
// fulfill the spec.
//
// This can be used if application-level validation detects value errors, to
// obtain a reasonable SourceRange to use for generated diagnostics. It works
// best when applied to specific body items (e.g. using AttrSpec, BlockSpec, ...)
// as opposed to entire bodies using ObjectSpec, TupleSpec. The result will
// be less useful the broader the specification, so e.g. a spec that returns
// the entirety of all of the blocks of a given type is likely to be
// _particularly_ arbitrary and useless.
//
// If the given body is not valid per the given spec, the result is best-effort
// and may not actually be something ideal. It's expected that an application
// will already have used Decode or PartialDecode earlier and thus had an
// opportunity to detect and report spec violations.
func SourceRange(body hcl.Body, spec Spec) hcl.Range {
	return sourceRange(body, nil, spec)
}
