package dynblock

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// expandBody wraps another hcl.Body and expands any "dynamic" blocks found
// inside whenever Content or PartialContent is called.
type expandBody struct {
	original   hcl.Body
	forEachCtx *hcl.EvalContext
	iteration  *iteration // non-nil if we're nested inside another "dynamic" block

	// These are used with PartialContent to produce a "remaining items"
	// body to return. They are nil on all bodies fresh out of the transformer.
	//
	// Note that this is re-implemented here rather than delegating to the
	// existing support required by the underlying body because we need to
	// retain access to the entire original body on subsequent decode operations
	// so we can retain any "dynamic" blocks for types we didn't take consume
	// on the first pass.
	hiddenAttrs  map[string]struct{}
	hiddenBlocks map[string]hcl.BlockHeaderSchema
}

func (b *expandBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	extSchema := b.extendSchema(schema)
	rawContent, diags := b.original.Content(extSchema)

	blocks, blockDiags := b.expandBlocks(schema, rawContent.Blocks, false)
	diags = append(diags, blockDiags...)
	attrs := b.prepareAttributes(rawContent.Attributes)

	content := &hcl.BodyContent{
		Attributes:       attrs,
		Blocks:           blocks,
		MissingItemRange: b.original.MissingItemRange(),
	}

	return content, diags
}

func (b *expandBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	extSchema := b.extendSchema(schema)
	rawContent, _, diags := b.original.PartialContent(extSchema)
	// We discard the "remain" argument above because we're going to construct
	// our own remain that also takes into account remaining "dynamic" blocks.

	blocks, blockDiags := b.expandBlocks(schema, rawContent.Blocks, true)
	diags = append(diags, blockDiags...)
	attrs := b.prepareAttributes(rawContent.Attributes)

	content := &hcl.BodyContent{
		Attributes:       attrs,
		Blocks:           blocks,
		MissingItemRange: b.original.MissingItemRange(),
	}

	remain := &expandBody{
		original:     b.original,
		forEachCtx:   b.forEachCtx,
		iteration:    b.iteration,
		hiddenAttrs:  make(map[string]struct{}),
		hiddenBlocks: make(map[string]hcl.BlockHeaderSchema),
	}
	for name := range b.hiddenAttrs {
		remain.hiddenAttrs[name] = struct{}{}
	}
	for typeName, blockS := range b.hiddenBlocks {
		remain.hiddenBlocks[typeName] = blockS
	}
	for _, attrS := range schema.Attributes {
		remain.hiddenAttrs[attrS.Name] = struct{}{}
	}
	for _, blockS := range schema.Blocks {
		remain.hiddenBlocks[blockS.Type] = blockS
	}

	return content, remain, diags
}

func (b *expandBody) extendSchema(schema *hcl.BodySchema) *hcl.BodySchema {
	// We augment the requested schema to also include our special "dynamic"
	// block type, since then we'll get instances of it interleaved with
	// all of the literal child blocks we must also include.
	extSchema := &hcl.BodySchema{
		Attributes: schema.Attributes,
		Blocks:     make([]hcl.BlockHeaderSchema, len(schema.Blocks), len(schema.Blocks)+len(b.hiddenBlocks)+1),
	}
	copy(extSchema.Blocks, schema.Blocks)
	extSchema.Blocks = append(extSchema.Blocks, dynamicBlockHeaderSchema)

	// If we have any hiddenBlocks then we also need to register those here
	// so that a call to "Content" on the underlying body won't fail.
	// (We'll filter these out again once we process the result of either
	// Content or PartialContent.)
	for _, blockS := range b.hiddenBlocks {
		extSchema.Blocks = append(extSchema.Blocks, blockS)
	}

	// If we have any hiddenAttrs then we also need to register these, for
	// the same reason as we deal with hiddenBlocks above.
	if len(b.hiddenAttrs) != 0 {
		newAttrs := make([]hcl.AttributeSchema, len(schema.Attributes), len(schema.Attributes)+len(b.hiddenAttrs))
		copy(newAttrs, extSchema.Attributes)
		for name := range b.hiddenAttrs {
			newAttrs = append(newAttrs, hcl.AttributeSchema{
				Name:     name,
				Required: false,
			})
		}
		extSchema.Attributes = newAttrs
	}

	return extSchema
}

func (b *expandBody) prepareAttributes(rawAttrs hcl.Attributes) hcl.Attributes {
	if len(b.hiddenAttrs) == 0 && b.iteration == nil {
		// Easy path: just pass through the attrs from the original body verbatim
		return rawAttrs
	}

	// Otherwise we have some work to do: we must filter out any attributes
	// that are hidden (since a previous PartialContent call already saw these)
	// and wrap the expressions of the inner attributes so that they will
	// have access to our iteration variables.
	attrs := make(hcl.Attributes, len(rawAttrs))
	for name, rawAttr := range rawAttrs {
		if _, hidden := b.hiddenAttrs[name]; hidden {
			continue
		}
		if b.iteration != nil {
			attr := *rawAttr // shallow copy so we can mutate it
			attr.Expr = exprWrap{
				Expression: attr.Expr,
				i:          b.iteration,
			}
			attrs[name] = &attr
		} else {
			// If we have no active iteration then no wrapping is required.
			attrs[name] = rawAttr
		}
	}
	return attrs
}

func (b *expandBody) expandBlocks(schema *hcl.BodySchema, rawBlocks hcl.Blocks, partial bool) (hcl.Blocks, hcl.Diagnostics) {
	var blocks hcl.Blocks
	var diags hcl.Diagnostics

	for _, rawBlock := range rawBlocks {
		switch rawBlock.Type {
		case "dynamic":
			realBlockType := rawBlock.Labels[0]
			if _, hidden := b.hiddenBlocks[realBlockType]; hidden {
				continue
			}

			var blockS *hcl.BlockHeaderSchema
			for _, candidate := range schema.Blocks {
				if candidate.Type == realBlockType {
					blockS = &candidate
					break
				}
			}
			if blockS == nil {
				// Not a block type that the caller requested.
				if !partial {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unsupported block type",
						Detail:   fmt.Sprintf("Blocks of type %q are not expected here.", realBlockType),
						Subject:  &rawBlock.LabelRanges[0],
					})
				}
				continue
			}

			spec, specDiags := b.decodeSpec(blockS, rawBlock)
			diags = append(diags, specDiags...)
			if specDiags.HasErrors() {
				continue
			}

			if spec.forEachVal.IsKnown() {
				for it := spec.forEachVal.ElementIterator(); it.Next(); {
					key, value := it.Element()
					i := b.iteration.MakeChild(spec.iteratorName, key, value)

					block, blockDiags := spec.newBlock(i, b.forEachCtx)
					diags = append(diags, blockDiags...)
					if block != nil {
						// Attach our new iteration context so that attributes
						// and other nested blocks can refer to our iterator.
						block.Body = b.expandChild(block.Body, i)
						blocks = append(blocks, block)
					}
				}
			} else {
				// If our top-level iteration value isn't known then we're forced
				// to compromise since HCL doesn't have any concept of an
				// "unknown block". In this case then, we'll produce a single
				// dynamic block with the iterator values set to DynamicVal,
				// which at least makes the potential for a block visible
				// in our result, even though it's not represented in a fully-accurate
				// way.
				i := b.iteration.MakeChild(spec.iteratorName, cty.DynamicVal, cty.DynamicVal)
				block, blockDiags := spec.newBlock(i, b.forEachCtx)
				diags = append(diags, blockDiags...)
				if block != nil {
					block.Body = b.expandChild(block.Body, i)

					// We additionally force all of the leaf attribute values
					// in the result to be unknown so the calling application
					// can, if necessary, use that as a heuristic to detect
					// when a single nested block might be standing in for
					// multiple blocks yet to be expanded. This retains the
					// structure of the generated body but forces all of its
					// leaf attribute values to be unknown.
					block.Body = unknownBody{block.Body}

					blocks = append(blocks, block)
				}
			}

		default:
			if _, hidden := b.hiddenBlocks[rawBlock.Type]; !hidden {
				// A static block doesn't create a new iteration context, but
				// it does need to inherit _our own_ iteration context in
				// case it contains expressions that refer to our inherited
				// iterators, or nested "dynamic" blocks.
				expandedBlock := *rawBlock // shallow copy
				expandedBlock.Body = b.expandChild(rawBlock.Body, b.iteration)
				blocks = append(blocks, &expandedBlock)
			}
		}
	}

	return blocks, diags
}

func (b *expandBody) expandChild(child hcl.Body, i *iteration) hcl.Body {
	chiCtx := i.EvalContext(b.forEachCtx)
	ret := Expand(child, chiCtx)
	ret.(*expandBody).iteration = i
	return ret
}

func (b *expandBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	// blocks aren't allowed in JustAttributes mode and this body can
	// only produce blocks, so we'll just pass straight through to our
	// underlying body here.
	return b.original.JustAttributes()
}

func (b *expandBody) MissingItemRange() hcl.Range {
	return b.original.MissingItemRange()
}
