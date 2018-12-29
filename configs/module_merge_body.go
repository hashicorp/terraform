package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// MergeBodies creates a new HCL body that contains a combination of the
// given base and override bodies. Attributes and blocks defined in the
// override body take precedence over those of the same name defined in
// the base body.
//
// If any block of a particular type appears in "override" then it will
// replace _all_ of the blocks of the same type in "base" in the new
// body.
func MergeBodies(base, override hcl.Body) hcl.Body {
	return mergeBody{
		Base:     base,
		Override: override,
	}
}

// mergeBody is a hcl.Body implementation that wraps a pair of other bodies
// and allows attributes and blocks within the override to take precedence
// over those defined in the base body.
//
// This is used to deal with dynamically-processed bodies in Module.mergeFile.
// It uses a shallow-only merging strategy where direct attributes defined
// in Override will override attributes of the same name in Base, while any
// blocks defined in Override will hide all blocks of the same type in Base.
//
// This cannot possibly "do the right thing" in all cases, because we don't
// have enough information about user intent. However, this behavior is intended
// to be reasonable for simple overriding use-cases.
type mergeBody struct {
	Base     hcl.Body
	Override hcl.Body
}

var _ hcl.Body = mergeBody{}

func (b mergeBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	oSchema := schemaForOverrides(schema)

	baseContent, cDiags := b.Base.Content(schema)
	diags = append(diags, cDiags...)
	overrideContent, cDiags := b.Override.Content(oSchema)
	diags = append(diags, cDiags...)

	content := b.prepareContent(baseContent, overrideContent)

	return content, diags
}

func (b mergeBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	oSchema := schemaForOverrides(schema)

	baseContent, baseRemain, cDiags := b.Base.PartialContent(schema)
	diags = append(diags, cDiags...)
	overrideContent, overrideRemain, cDiags := b.Override.PartialContent(oSchema)
	diags = append(diags, cDiags...)

	content := b.prepareContent(baseContent, overrideContent)

	remain := MergeBodies(baseRemain, overrideRemain)

	return content, remain, diags
}

func (b mergeBody) prepareContent(base *hcl.BodyContent, override *hcl.BodyContent) *hcl.BodyContent {
	content := &hcl.BodyContent{
		Attributes: make(hcl.Attributes),
	}

	// For attributes we just assign from each map in turn and let the override
	// map clobber any matching entries from base.
	for k, a := range base.Attributes {
		content.Attributes[k] = a
	}
	for k, a := range override.Attributes {
		content.Attributes[k] = a
	}

	// Things are a little more interesting for blocks because they arrive
	// as a flat list. Our merging semantics call for us to suppress blocks
	// from base if at least one block of the same type appears in override.
	// We explicitly do not try to correlate and deeply merge nested blocks,
	// since we don't have enough context here to infer user intent.

	overriddenBlockTypes := make(map[string]bool)
	for _, block := range override.Blocks {
		overriddenBlockTypes[block.Type] = true
	}
	for _, block := range base.Blocks {
		if overriddenBlockTypes[block.Type] {
			continue
		}
		content.Blocks = append(content.Blocks, block)
	}
	for _, block := range override.Blocks {
		content.Blocks = append(content.Blocks, block)
	}

	return content
}

func (b mergeBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := make(hcl.Attributes)

	baseAttrs, aDiags := b.Base.JustAttributes()
	diags = append(diags, aDiags...)
	overrideAttrs, aDiags := b.Override.JustAttributes()
	diags = append(diags, aDiags...)

	for k, a := range baseAttrs {
		ret[k] = a
	}
	for k, a := range overrideAttrs {
		ret[k] = a
	}

	return ret, diags
}

func (b mergeBody) MissingItemRange() hcl.Range {
	return b.Base.MissingItemRange()
}
