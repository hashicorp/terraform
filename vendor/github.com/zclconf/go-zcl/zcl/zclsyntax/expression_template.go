package zclsyntax

import (
	"bytes"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-zcl/zcl"
)

type TemplateExpr struct {
	Parts []Expression

	SrcRange zcl.Range
}

func (e *TemplateExpr) walkChildNodes(w internalWalkFunc) {
	for i, part := range e.Parts {
		e.Parts[i] = w(part).(Expression)
	}
}

func (e *TemplateExpr) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	buf := &bytes.Buffer{}
	var diags zcl.Diagnostics
	isKnown := true

	for _, part := range e.Parts {
		partVal, partDiags := part.Value(ctx)
		diags = append(diags, partDiags...)

		if partVal.IsNull() {
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Invalid template interpolation value",
				Detail: fmt.Sprintf(
					"The expression result is null. Cannot include a null value in a string template.",
				),
				Subject: part.Range().Ptr(),
				Context: &e.SrcRange,
			})
			continue
		}

		if !partVal.IsKnown() {
			// If any part is unknown then the result as a whole must be
			// unknown too. We'll keep on processing the rest of the parts
			// anyway, because we want to still emit any diagnostics resulting
			// from evaluating those.
			isKnown = false
			continue
		}

		strVal, err := convert.Convert(partVal, cty.String)
		if err != nil {
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Invalid template interpolation value",
				Detail: fmt.Sprintf(
					"Cannot include the given value in a string template: %s.",
					err.Error(),
				),
				Subject: part.Range().Ptr(),
				Context: &e.SrcRange,
			})
			continue
		}

		buf.WriteString(strVal.AsString())
	}

	if !isKnown {
		return cty.UnknownVal(cty.String), diags
	}

	return cty.StringVal(buf.String()), diags
}

func (e *TemplateExpr) Range() zcl.Range {
	return e.SrcRange
}

func (e *TemplateExpr) StartRange() zcl.Range {
	return e.Parts[0].StartRange()
}

// TemplateJoinExpr is used to convert tuples of strings produced by template
// constructs (i.e. for loops) into flat strings, by converting the values
// tos strings and joining them. This AST node is not used directly; it's
// produced as part of the AST of a "for" loop in a template.
type TemplateJoinExpr struct {
	Tuple Expression
}

func (e *TemplateJoinExpr) walkChildNodes(w internalWalkFunc) {
	e.Tuple = w(e.Tuple).(Expression)
}

func (e *TemplateJoinExpr) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	tuple, diags := e.Tuple.Value(ctx)

	if tuple.IsNull() {
		// This indicates a bug in the code that constructed the AST.
		panic("TemplateJoinExpr got null tuple")
	}
	if tuple.Type() == cty.DynamicPseudoType {
		return cty.UnknownVal(cty.String), diags
	}
	if !tuple.Type().IsTupleType() {
		// This indicates a bug in the code that constructed the AST.
		panic("TemplateJoinExpr got non-tuple tuple")
	}
	if !tuple.IsKnown() {
		return cty.UnknownVal(cty.String), diags
	}

	buf := &bytes.Buffer{}
	it := tuple.ElementIterator()
	for it.Next() {
		_, val := it.Element()

		if val.IsNull() {
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Invalid template interpolation value",
				Detail: fmt.Sprintf(
					"An iteration result is null. Cannot include a null value in a string template.",
				),
				Subject: e.Range().Ptr(),
			})
			continue
		}
		if val.Type() == cty.DynamicPseudoType {
			return cty.UnknownVal(cty.String), diags
		}
		strVal, err := convert.Convert(val, cty.String)
		if err != nil {
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Invalid template interpolation value",
				Detail: fmt.Sprintf(
					"Cannot include one of the interpolation results into the string template: %s.",
					err.Error(),
				),
				Subject: e.Range().Ptr(),
			})
			continue
		}
		if !val.IsKnown() {
			return cty.UnknownVal(cty.String), diags
		}

		buf.WriteString(strVal.AsString())
	}

	return cty.StringVal(buf.String()), diags
}

func (e *TemplateJoinExpr) Range() zcl.Range {
	return e.Tuple.Range()
}

func (e *TemplateJoinExpr) StartRange() zcl.Range {
	return e.Tuple.StartRange()
}

// TemplateWrapExpr is used instead of a TemplateExpr when a template
// consists _only_ of a single interpolation sequence. In that case, the
// template's result is the single interpolation's result, verbatim with
// no type conversions.
type TemplateWrapExpr struct {
	Wrapped Expression

	SrcRange zcl.Range
}

func (e *TemplateWrapExpr) walkChildNodes(w internalWalkFunc) {
	e.Wrapped = w(e.Wrapped).(Expression)
}

func (e *TemplateWrapExpr) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	return e.Wrapped.Value(ctx)
}

func (e *TemplateWrapExpr) Range() zcl.Range {
	return e.SrcRange
}

func (e *TemplateWrapExpr) StartRange() zcl.Range {
	return e.SrcRange
}
