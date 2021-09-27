package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// lintExpressionGeneric runs a generic set of linting rules on the given
// expression, which we assume is just a normal expression that'll be evaluated
// for its value in a lang.Scope, but with no specific assumptions about
// where in the configuration it is placed.
//
// It's safe to pass a nil expression to this function, in which case this
// function will never produce any lint warnings.
func lintExpressionGeneric(expr hcl.Expression) tfdiags.Diagnostics {
	if expr == nil {
		return nil
	}

	// No individual expression rules yet
	return nil
}

// lintExpressionsInBodyGeneric runs lintExpressionGeneric against each of
// the expressions detected inside the given body, which it detects using
// the given schema.
//
// If the body doesn't fully conform to the schema then
// lintExpressionsInBodyGeneric will skip over some or all of the body contents
// in order to still produce a partial result where possible.
//
// It's safe to pass a nil body or schema to this function, in which case this
// function will never produce any lint warnings.
func lintExpressionsInBodyGeneric(body hcl.Body, schema *configschema.Block) tfdiags.Diagnostics {
	if body == nil || schema == nil {
		return nil
	}

	return visitExpressionsInBody(body, schema, lintExpressionGeneric)
}

// visitExpressionsInBody is a helper for writting linter-like logic which
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
