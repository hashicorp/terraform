package hcldec

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

func decode(body hcl.Body, block *hcl.Block, ctx *hcl.EvalContext, spec Spec, partial bool) (cty.Value, hcl.Body, hcl.Diagnostics) {
	schema := ImpliedSchema(spec)

	var content *hcl.BodyContent
	var diags hcl.Diagnostics
	var leftovers hcl.Body

	if partial {
		content, leftovers, diags = body.PartialContent(schema)
	} else {
		content, diags = body.Content(schema)
	}

	val, valDiags := spec.decode(content, block, ctx)
	diags = append(diags, valDiags...)

	return val, leftovers, diags
}

func sourceRange(body hcl.Body, block *hcl.Block, spec Spec) hcl.Range {
	schema := ImpliedSchema(spec)
	content, _, _ := body.PartialContent(schema)

	return spec.sourceRange(content, block)
}
