// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"github.com/hashicorp/hcl/v2"
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
	baseSchema := schemaWithDynamic(schema)
	overrideSchema := schemaWithDynamic(schemaForOverrides(schema))

	baseContent, _, cDiags := b.Base.PartialContent(baseSchema)
	diags = append(diags, cDiags...)
	overrideContent, _, cDiags := b.Override.PartialContent(overrideSchema)
	diags = append(diags, cDiags...)

	content := b.prepareContent(baseContent, overrideContent)

	return content, diags
}

func (b mergeBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	baseSchema := schemaWithDynamic(schema)
	overrideSchema := schemaWithDynamic(schemaForOverrides(schema))

	baseContent, baseRemain, cDiags := b.Base.PartialContent(baseSchema)
	diags = append(diags, cDiags...)
	overrideContent, overrideRemain, cDiags := b.Override.PartialContent(overrideSchema)
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
		if block.Type == "dynamic" {
			overriddenBlockTypes[block.Labels[0]] = true
			continue
		}
		overriddenBlockTypes[block.Type] = true
	}
	for _, block := range base.Blocks {
		// We skip over dynamic blocks whose type label is an overridden type
		// but note that below we do still leave them as dynamic blocks in
		// the result because expanding the dynamic blocks that are left is
		// done much later during the core graph walks, where we can safely
		// evaluate the expressions.
		if block.Type == "dynamic" && overriddenBlockTypes[block.Labels[0]] {
			continue
		}
		if overriddenBlockTypes[block.Type] {
			continue
		}
		content.Blocks = append(content.Blocks, block)
	}
	content.Blocks = append(content.Blocks, override.Blocks...)

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
