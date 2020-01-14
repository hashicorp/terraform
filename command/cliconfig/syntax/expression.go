package syntax

import (
	"fmt"
	"os"

	hcl1token "github.com/hashicorp/hcl/hcl/token"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/tfdiags"
)

// ExpandExpression is an hcl.Expression implementation that accepts the
// variable syntax defined by os.Expand on any string that appears inside its
// value.
//
// Note that it currently supports only direct strings and slices of strings,
// because those are the only situations currently used in the CLI config
// language. To use environment variable expansion in other constructs will
// require first expanding the implementation to include other types.
type ExpandExpression struct {
	raw interface{}
	pos hcl1token.Pos
}

var _ hcl2.Expression = (*ExpandExpression)(nil)

// Value implements hcl.Expression.Value by calling os.Expand on any string
// appearing inside the expression value, assuming that the variables in the
// given context are environment variables.
func (e *ExpandExpression) Value(ctx *hcl2.EvalContext) (cty.Value, hcl2.Diagnostics) {
	switch raw := e.raw.(type) {
	case string:
		return cty.StringVal(expandFromEvalContext(raw, ctx)), nil
	case []string:
		vals := make([]cty.Value, len(raw))
		for i, s := range raw {
			vals[i] = cty.StringVal(expandFromEvalContext(s, ctx))
		}
		return cty.ListVal(vals), nil
	default:
		var diags hcl2.Diagnostics
		// FIXME: We don't have enough context here to produce a good error
		// message, because we don't know if the caller wants a string or a
		// list of string. We know it's one or the other, but telling that to
		// the user would be confusing because the user must still know which
		// one in order to fix it.
		diags = diags.Append(&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Unsupported value type",
			Detail:   "This value is of the wrong type for this CLI configuration argument.",
			Subject:  hcl1PosAsHCL2Range(e.pos).Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

// Variables always returns an empty set of traversals, contrary to the
// definition of hcl.Expression.Variables, because we know that in practice
// the CLI configuration decoder never needs to inspect references prior to
// evaluation.
func (e *ExpandExpression) Variables() []hcl2.Traversal {
	return nil
}

func expandFromEvalContext(str string, ctx *hcl2.EvalContext) string {
	return os.Expand(str, func(name string) string {
		v, ok := ctx.Variables[name]
		if !ok {
			return ""
		}
		if v.Type() != cty.String || v.IsNull() || !v.IsKnown() {
			// Should never happen, because CLI Config only supports environment
			// variables as referenceable values and they are always known strings.
			return ""
		}
		return v.AsString()
	})
}

// Range implements hcl.Expression.Range, returning an approximate range
// derived from the underlying HCL 1 value.
func (e *ExpandExpression) Range() hcl2.Range {
	return hcl1PosAsHCL2Range(e.pos)
}

// StartRange is an alias for Range.
func (e *ExpandExpression) StartRange() hcl2.Range {
	return e.Range()
}

func literalExpr(raw interface{}, pos hcl1token.Pos) (hcl2.Expression, hcl2.Diagnostics) {
	ty, err := gocty.ImpliedType(raw)
	if err != nil {
		var diags hcl2.Diagnostics
		diags = diags.Append(&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid argument value",
			Detail:   fmt.Sprintf("Cannot decode this argument value: %s.", tfdiags.FormatError(err)),
			Subject:  hcl1PosAsHCL2Range(pos).Ptr(),
		})
		return hcl2.StaticExpr(cty.DynamicVal, hcl1PosAsHCL2Range(pos)), diags
	}

	val, err := gocty.ToCtyValue(raw, ty)
	if err != nil {
		var diags hcl2.Diagnostics
		diags = diags.Append(&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid argument value",
			Detail:   fmt.Sprintf("Cannot decode this argument value: %s.", tfdiags.FormatError(err)),
			Subject:  hcl1PosAsHCL2Range(pos).Ptr(),
		})
		return hcl2.StaticExpr(cty.DynamicVal, hcl1PosAsHCL2Range(pos)), diags
	}

	return hcl2.StaticExpr(val, hcl1PosAsHCL2Range(pos)), nil
}
