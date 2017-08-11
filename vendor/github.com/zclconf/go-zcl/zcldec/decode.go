package zcldec

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-zcl/zcl"
)

func decode(body zcl.Body, block *zcl.Block, ctx *zcl.EvalContext, spec Spec, partial bool) (cty.Value, zcl.Body, zcl.Diagnostics) {
	schema := ImpliedSchema(spec)

	var content *zcl.BodyContent
	var diags zcl.Diagnostics
	var leftovers zcl.Body

	if partial {
		content, leftovers, diags = body.PartialContent(schema)
	} else {
		content, diags = body.Content(schema)
	}

	val, valDiags := spec.decode(content, block, ctx)
	diags = append(diags, valDiags...)

	return val, leftovers, diags
}

func sourceRange(body zcl.Body, block *zcl.Block, spec Spec) zcl.Range {
	schema := ImpliedSchema(spec)
	content, _, _ := body.PartialContent(schema)

	return spec.sourceRange(content, block)
}
