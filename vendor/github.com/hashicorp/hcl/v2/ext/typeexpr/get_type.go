package typeexpr

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

const invalidTypeSummary = "Invalid type specification"

// getType is the internal implementation of both Type and TypeConstraint,
// using the passed flag to distinguish. When constraint is false, the "any"
// keyword will produce an error.
func getType(expr hcl.Expression, constraint bool) (cty.Type, hcl.Diagnostics) {
	// First we'll try for one of our keywords
	kw := hcl.ExprAsKeyword(expr)
	switch kw {
	case "bool":
		return cty.Bool, nil
	case "string":
		return cty.String, nil
	case "number":
		return cty.Number, nil
	case "any":
		if constraint {
			return cty.DynamicPseudoType, nil
		}
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("The keyword %q cannot be used in this type specification: an exact type is required.", kw),
			Subject:  expr.Range().Ptr(),
		}}
	case "list", "map", "set":
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("The %s type constructor requires one argument specifying the element type.", kw),
			Subject:  expr.Range().Ptr(),
		}}
	case "object":
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "The object type constructor requires one argument specifying the attribute types and values as a map.",
			Subject:  expr.Range().Ptr(),
		}}
	case "tuple":
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "The tuple type constructor requires one argument specifying the element types as a list.",
			Subject:  expr.Range().Ptr(),
		}}
	case "":
		// okay! we'll fall through and try processing as a call, then.
	default:
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("The keyword %q is not a valid type specification.", kw),
			Subject:  expr.Range().Ptr(),
		}}
	}

	// If we get down here then our expression isn't just a keyword, so we'll
	// try to process it as a call instead.
	call, diags := hcl.ExprCall(expr)
	if diags.HasErrors() {
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "A type specification is either a primitive type keyword (bool, number, string) or a complex type constructor call, like list(string).",
			Subject:  expr.Range().Ptr(),
		}}
	}

	switch call.Name {
	case "bool", "string", "number", "any":
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Primitive type keyword %q does not expect arguments.", call.Name),
			Subject:  &call.ArgsRange,
		}}
	}

	if len(call.Arguments) != 1 {
		contextRange := call.ArgsRange
		subjectRange := call.ArgsRange
		if len(call.Arguments) > 1 {
			// If we have too many arguments (as opposed to too _few_) then
			// we'll highlight the extraneous arguments as the diagnostic
			// subject.
			subjectRange = hcl.RangeBetween(call.Arguments[1].Range(), call.Arguments[len(call.Arguments)-1].Range())
		}

		switch call.Name {
		case "list", "set", "map":
			return cty.DynamicPseudoType, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   fmt.Sprintf("The %s type constructor requires one argument specifying the element type.", call.Name),
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		case "object":
			return cty.DynamicPseudoType, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The object type constructor requires one argument specifying the attribute types and values as a map.",
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		case "tuple":
			return cty.DynamicPseudoType, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The tuple type constructor requires one argument specifying the element types as a list.",
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		}
	}

	switch call.Name {

	case "list":
		ety, diags := getType(call.Arguments[0], constraint)
		return cty.List(ety), diags
	case "set":
		ety, diags := getType(call.Arguments[0], constraint)
		return cty.Set(ety), diags
	case "map":
		ety, diags := getType(call.Arguments[0], constraint)
		return cty.Map(ety), diags
	case "object":
		attrDefs, diags := hcl.ExprMap(call.Arguments[0])
		if diags.HasErrors() {
			return cty.DynamicPseudoType, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "Object type constructor requires a map whose keys are attribute names and whose values are the corresponding attribute types.",
				Subject:  call.Arguments[0].Range().Ptr(),
				Context:  expr.Range().Ptr(),
			}}
		}

		atys := make(map[string]cty.Type)
		for _, attrDef := range attrDefs {
			attrName := hcl.ExprAsKeyword(attrDef.Key)
			if attrName == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  invalidTypeSummary,
					Detail:   "Object constructor map keys must be attribute names.",
					Subject:  attrDef.Key.Range().Ptr(),
					Context:  expr.Range().Ptr(),
				})
				continue
			}
			aty, attrDiags := getType(attrDef.Value, constraint)
			diags = append(diags, attrDiags...)
			atys[attrName] = aty
		}
		return cty.Object(atys), diags
	case "tuple":
		elemDefs, diags := hcl.ExprList(call.Arguments[0])
		if diags.HasErrors() {
			return cty.DynamicPseudoType, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "Tuple type constructor requires a list of element types.",
				Subject:  call.Arguments[0].Range().Ptr(),
				Context:  expr.Range().Ptr(),
			}}
		}
		etys := make([]cty.Type, len(elemDefs))
		for i, defExpr := range elemDefs {
			ety, elemDiags := getType(defExpr, constraint)
			diags = append(diags, elemDiags...)
			etys[i] = ety
		}
		return cty.Tuple(etys), diags
	default:
		// Can't access call.Arguments in this path because we've not validated
		// that it contains exactly one expression here.
		return cty.DynamicPseudoType, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Keyword %q is not a valid type constructor.", call.Name),
			Subject:  expr.Range().Ptr(),
		}}
	}
}
