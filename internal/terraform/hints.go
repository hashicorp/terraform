package terraform

import (
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// hintExpressionGeneric runs a generic set of hinting rules on the given
// expression, which we assume is just a normal expression that'll be evaluated
// for its value in a lang.Scope, but with no specific assumptions about
// where in the configuration it is placed.
//
// It's safe to pass a nil expression to this function, in which case this
// function will never produce any hints.
func hintExpressionGeneric(expr hcl.Expression) tfdiags.Diagnostics {
	// We can only do deep static analysis on native syntax expressions.
	nativeExpr, ok := expr.(hclsyntax.Expression)
	if !ok {
		return nil
	}

	// hint functions are only allowed to produce *tfdiags.HintMessage diagnostics
	var diags tfdiags.Diagnostics

	hclsyntax.VisitAll(nativeExpr, func(node hclsyntax.Node) hcl.Diagnostics {
		log.Printf("[TRACE] hintExpressionGeneric visiting %T", node)

		switch expr := node.(type) {

		case *hclsyntax.IndexExpr:
			log.Printf("[TRACE] It's an index expression, with %T!", expr.Collection)

			if splat, ok := expr.Collection.(*hclsyntax.SplatExpr); ok {
				// Indexing a splat result of a sequence with a number is
				// typically better written as a plain index expression.
				// However, we'll only warn this if we can prove that the
				// key is a number.
				keyExpr := expr.Key
				val, valDiags := keyExpr.Value(&hcl.EvalContext{
					Variables: map[string]cty.Value{
						"count": cty.ObjectVal(map[string]cty.Value{
							"index": cty.UnknownVal(cty.Number),
						}),
					},
				})
				if !valDiags.HasErrors() && val.Type() == cty.Number {
					diags = diags.Append(&tfdiags.HintMessage{
						// TODO: This message should include a real example of
						// how to rewrite the given expression, but to achieve
						// that we'd either need access to the original source
						// code so we can slice it up, or a way to turn
						// hclsyntax AST nodes back into equivalent source code.
						Summary:     "Unnecessary splat expression with index",
						Detail:      "Looking up a particular index of a splat expression result is the same as just directly using the index instead of the splat operator.",
						SourceRange: tfdiags.SourceRangeFromHCL(splat.MarkerRange),
					})
				}
			}

		default:
			// Nothing to do for any other types.
		}

		return nil
	})

	// No individual expression rules yet
	return diags
}

// hintExpressionsInBodyGeneric runs hintExpressionGeneric against each of
// the expressions detected inside the given body, which it detects using
// the given schema.
//
// If the body doesn't fully conform to the schema then
// hintExpressionsInBodyGeneric will skip over some or all of the body contents
// in order to still produce a partial result where possible.
//
// It's safe to pass a nil body or schema to this function, in which case this
// function will never produce any hints.
func hintExpressionsInBodyGeneric(body hcl.Body, schema *configschema.Block) tfdiags.Diagnostics {
	if body == nil || schema == nil {
		return nil
	}

	return visitExpressionsInBody(body, schema, hintExpressionGeneric)
}

// visitExpressionsInBody is a helper for writting hinter-like logic which
// applies to all expressions declared in a particular HCL body, which should
// conform to the given schema.
//
// If the body doesn't fully conform to the schema then visitExpressionsInBody
// will skip over some or all of the body contents in order to still produce
// a partial result. Therefore this expression isn't suitable for implementing
// blocking validation, because it only makes a best effort to find expressions.
func visitExpressionsInBody(body hcl.Body, schema *configschema.Block, cb func(hcl.Expression) tfdiags.Diagnostics) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	var hclSchema hcl.BodySchema
	for name := range schema.Attributes {
		hclSchema.Attributes = append(hclSchema.Attributes, hcl.AttributeSchema{
			Name: name,
			// We intentionally don't set "Required" here because we still
			// want to scan all given expressions, even if the author omitted
			// some arguments.
		})
	}
	for typeName, blockS := range schema.BlockTypes {
		var labelNames []string
		if blockS.Nesting == configschema.NestingMap {
			labelNames = []string{"key"}
		}

		hclSchema.Blocks = append(hclSchema.Blocks, hcl.BlockHeaderSchema{
			Type:       typeName,
			LabelNames: labelNames,
		})
	}

	content, _, _ := body.PartialContent(&hclSchema)

	for _, attr := range content.Attributes {
		diags = diags.Append(cb(attr.Expr))
	}

	for _, block := range content.Blocks {
		typeName := block.Type
		blockS, ok := schema.BlockTypes[typeName]
		if !ok {
			continue // Weird, but we'll tolerate it to be robust
		}
		diags = diags.Append(visitExpressionsInBody(block.Body, &blockS.Block, cb))
	}

	return diags
}
