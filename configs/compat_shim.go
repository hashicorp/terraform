package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// -------------------------------------------------------------------------
// Functions in this file are compatibility shims intended to ease conversion
// from the old configuration loader. Any use of these functions that makes
// a change should generate a deprecation warning explaining to the user how
// to update their code for new patterns.
//
// Shims are particularly important for any patterns that have been widely
// documented in books, tutorials, etc. Users will still be starting from
// these examples and we want to help them adopt the latest patterns rather
// than leave them stranded.
// -------------------------------------------------------------------------

// shimTraversalInString takes any arbitrary expression and checks if it is
// a quoted string in the native syntax. If it _is_, then it is parsed as a
// traversal and re-wrapped into a synthetic traversal expression and a
// warning is generated. Otherwise, the given expression is just returned
// verbatim.
//
// This function has no effect on expressions from the JSON syntax, since
// traversals in strings are the required pattern in that syntax.
//
// If wantKeyword is set, the generated warning diagnostic will talk about
// keywords rather than references. The behavior is otherwise unchanged, and
// the caller remains responsible for checking that the result is indeed
// a keyword, e.g. using hcl.ExprAsKeyword.
func shimTraversalInString(expr hcl.Expression, wantKeyword bool) (hcl.Expression, hcl.Diagnostics) {
	// ObjectConsKeyExpr is a special wrapper type used for keys on object
	// constructors to deal with the fact that naked identifiers are normally
	// handled as "bareword" strings rather than as variable references. Since
	// we know we're interpreting as a traversal anyway (and thus it won't
	// matter whether it's a string or an identifier) we can safely just unwrap
	// here and then process whatever we find inside as normal.
	if ocke, ok := expr.(*hclsyntax.ObjectConsKeyExpr); ok {
		expr = ocke.Wrapped
	}

	if !exprIsNativeQuotedString(expr) {
		return expr, nil
	}

	strVal, diags := expr.Value(nil)
	if diags.HasErrors() || strVal.IsNull() || !strVal.IsKnown() {
		// Since we're not even able to attempt a shim here, we'll discard
		// the diagnostics we saw so far and let the caller's own error
		// handling take care of reporting the invalid expression.
		return expr, nil
	}

	// The position handling here isn't _quite_ right because it won't
	// take into account any escape sequences in the literal string, but
	// it should be close enough for any error reporting to make sense.
	srcRange := expr.Range()
	startPos := srcRange.Start // copy
	startPos.Column++          // skip initial quote
	startPos.Byte++            // skip initial quote

	traversal, tDiags := hclsyntax.ParseTraversalAbs(
		[]byte(strVal.AsString()),
		srcRange.Filename,
		startPos,
	)
	diags = append(diags, tDiags...)

	if wantKeyword {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Quoted keywords are deprecated",
			Detail:   "In this context, keywords are expected literally rather than in quotes. Terraform 0.11 and earlier required quotes, but quoted keywords are now deprecated and will be removed in a future version of Terraform. Remove the quotes surrounding this keyword to silence this warning.",
			Subject:  &srcRange,
		})
	} else {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Quoted references are deprecated",
			Detail:   "In this context, references are expected literally rather than in quotes. Terraform 0.11 and earlier required quotes, but quoted references are now deprecated and will be removed in a future version of Terraform. Remove the quotes surrounding this reference to silence this warning.",
			Subject:  &srcRange,
		})
	}

	return &hclsyntax.ScopeTraversalExpr{
		Traversal: traversal,
		SrcRange:  srcRange,
	}, diags
}

// shimIsIgnoreChangesStar returns true if the given expression seems to be
// a string literal whose value is "*". This is used to support a legacy
// form of ignore_changes = all .
//
// This function does not itself emit any diagnostics, so it's the caller's
// responsibility to emit a warning diagnostic when this function returns true.
func shimIsIgnoreChangesStar(expr hcl.Expression) bool {
	val, valDiags := expr.Value(nil)
	if valDiags.HasErrors() {
		return false
	}
	if val.Type() != cty.String || val.IsNull() || !val.IsKnown() {
		return false
	}
	return val.AsString() == "*"
}

// warnForDeprecatedInterpolations returns warning diagnostics if the given
// body can be proven to contain attributes whose expressions are native
// syntax expressions consisting entirely of a single template interpolation,
// which is a deprecated way to include a non-literal value in configuration.
//
// This is a best-effort sort of thing which relies on the physical HCL native
// syntax AST, so it might not catch everything. The main goal is to catch the
// "obvious" cases in order to help spread awareness that this old form is
// deprecated, when folks copy it from older examples they've found on the
// internet that were written for Terraform 0.11 or earlier.
func warnForDeprecatedInterpolationsInBody(body hcl.Body) hcl.Diagnostics {
	var diags hcl.Diagnostics

	nativeBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// If it's not native syntax then we've nothing to do here.
		return diags
	}

	for _, attr := range nativeBody.Attributes {
		moreDiags := warnForDeprecatedInterpolationsInExpr(attr.Expr)
		diags = append(diags, moreDiags...)
	}

	for _, block := range nativeBody.Blocks {
		// We'll also go hunting in nested blocks
		moreDiags := warnForDeprecatedInterpolationsInBody(block.Body)
		diags = append(diags, moreDiags...)
	}

	return diags
}

func warnForDeprecatedInterpolationsInExpr(expr hcl.Expression) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if _, ok := expr.(*hclsyntax.TemplateWrapExpr); !ok {
		// We're only interested in TemplateWrapExpr, because that's how
		// the HCL native syntax parser represents the case of a template
		// that consists entirely of a single interpolation expression, which
		// is therefore subject to the special case of passing through the
		// inner value without conversion to string.
		return diags
	}

	diags = append(diags, &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Interpolation-only expressions are deprecated",
		Detail:   "Terraform 0.11 and earlier required all non-constant expressions to be provided via interpolation syntax, but this pattern is now deprecated. To silence this warning, remove the \"${ sequence from the start and the }\" sequence from the end of this expression, leaving just the inner expression.\n\nTemplate interpolation syntax is still used to construct strings from expressions when the template includes multiple interpolation sequences or a mixture of literal strings and interpolations. This deprecation applies only to templates that consist entirely of a single interpolation sequence.",
		Subject:  expr.Range().Ptr(),
	})

	return diags
}
