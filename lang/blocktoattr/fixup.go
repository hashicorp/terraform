package blocktoattr

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// FixUpBlockAttrs takes a raw HCL body and adds some additional normalization
// functionality to allow attributes that are specified as having list or set
// type in the schema to be written with HCL block syntax as multiple nested
// blocks with the attribute name as the block type.
//
// This partially restores some of the block/attribute confusion from HCL 1
// so that existing patterns that depended on that confusion can continue to
// be used in the short term while we settle on a longer-term strategy.
//
// Most of the fixup work is actually done when the returned body is
// subsequently decoded, so while FixUpBlockAttrs always succeeds, the eventual
// decode of the body might not, if the content of the body is so ambiguous
// that there's no safe way to map it to the schema.
func FixUpBlockAttrs(body hcl.Body, schema *configschema.Block) hcl.Body {
	// The schema should never be nil, but in practice it seems to be sometimes
	// in the presence of poorly-configured test mocks, so we'll be robust
	// by synthesizing an empty one.
	if schema == nil {
		schema = &configschema.Block{}
	}

	// We'll do a quick sniff first to see if there's even anything ambiguous
	// in this schema. (We still need to wrap it even if not, just in case we
	// need to do fixup inside nested blocks.
	ambiguousNames := make(map[string]struct{})
	for name, attrS := range schema.Attributes {
		aty := attrS.Type
		if (aty.IsListType() || aty.IsSetType()) && aty.ElementType().IsObjectType() {
			ambiguousNames[name] = struct{}{}
		}
	}

	return &fixupBody{
		original: body,
		schema:   schema,
		names:    ambiguousNames,
	}
}

type fixupBody struct {
	original hcl.Body
	schema   *configschema.Block
	names    map[string]struct{}
}

// Content decodes content from the body. The given schema must be the lower-level
// representation of the same schema that was previously passed to FixUpBlockAttrs,
// or else the result is undefined.
func (b *fixupBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	schema = b.effectiveSchema(schema)
	content, diags := b.original.Content(schema)
	return b.fixupContent(content), diags
}

func (b *fixupBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	schema = b.effectiveSchema(schema)
	content, remain, diags := b.original.PartialContent(schema)
	remain = &fixupBody{
		original: remain,
		schema:   b.schema,
		names:    b.names,
	}
	return b.fixupContent(content), remain, diags
}

func (b *fixupBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	// FixUpBlockAttrs is not intended to be used in situations where we'd use
	// JustAttributes, so we just pass this through verbatim to complete our
	// implementation of hcl.Body.
	return b.original.JustAttributes()
}

func (b *fixupBody) MissingItemRange() hcl.Range {
	return b.original.MissingItemRange()
}

// effectiveSchema produces a derived *hcl.BodySchema by sniffing the body's
// content to determine whether the author has used attribute or block syntax
// for each of the ambigious attributes where both are permitted.
//
// The resulting schema will always contain all of the same names that are
// in the given schema, but some attribute schemas may instead be replaced by
// block header schemas.
func (b *fixupBody) effectiveSchema(given *hcl.BodySchema) *hcl.BodySchema {
	ret := &hcl.BodySchema{}

	appearsAsBlock := make(map[string]struct{})
	{
		// We'll construct some throwaway schemas here just to probe for
		// whether each of our ambiguous names seems to be being used as
		// an attribute or a block. We need to check both because in JSON
		// syntax we rely on the schema to decide between attribute or block
		// interpretation and so JSON will always answer yes to both of
		// these questions and we want to prefer the attribute interpretation
		// in that case.
		var probeSchema hcl.BodySchema

		for name := range b.names {
			probeSchema = hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: name,
					},
				},
			}
			content, _, _ := b.original.PartialContent(&probeSchema)
			if _, exists := content.Attributes[name]; exists {
				// Can decode as an attribute, so we'll go with that.
				continue
			}
			probeSchema = hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type: name,
					},
				},
			}
			content, _, _ = b.original.PartialContent(&probeSchema)
			if len(content.Blocks) > 0 {
				// No attribute present and at least one block present, so
				// we'll need to rewrite this one as a block for a successful
				// result.
				appearsAsBlock[name] = struct{}{}
			}
		}
	}

	for _, attrS := range given.Attributes {
		if _, exists := appearsAsBlock[attrS.Name]; exists {
			ret.Blocks = append(ret.Blocks, hcl.BlockHeaderSchema{
				Type: attrS.Name,
			})
		} else {
			ret.Attributes = append(ret.Attributes, attrS)
		}
	}

	// Anything that is specified as a block type in the input schema remains
	// that way by just passing through verbatim.
	ret.Blocks = append(ret.Blocks, given.Blocks...)

	return ret
}

func (b *fixupBody) fixupContent(content *hcl.BodyContent) *hcl.BodyContent {
	var ret hcl.BodyContent
	ret.Attributes = make(hcl.Attributes)
	for name, attr := range content.Attributes {
		ret.Attributes[name] = attr
	}
	blockAttrVals := make(map[string][]*hcl.Block)
	for _, block := range content.Blocks {
		if _, exists := b.names[block.Type]; exists {
			// If we get here then we've found a block type whose instances need
			// to be re-interpreted as a list-of-objects attribute. We'll gather
			// those up and fix them up below.
			blockAttrVals[block.Type] = append(blockAttrVals[block.Type], block)
			continue
		}

		// We need to now re-wrap our inner body so it will be subject to the
		// same attribute-as-block fixup when recursively decoded.
		retBlock := *block // shallow copy
		if blockS, ok := b.schema.BlockTypes[block.Type]; ok {
			// Would be weird if not ok, but we'll allow it for robustness; body just won't be fixed up, then
			retBlock.Body = FixUpBlockAttrs(retBlock.Body, &blockS.Block)
		}

		ret.Blocks = append(ret.Blocks, &retBlock)
	}
	// No we'll install synthetic attributes for each of our fixups. We can't
	// do this exactly because HCL's information model expects an attribute
	// to be a single decl but we have multiple separate blocks. We'll
	// approximate things, then, by using only our first block for the source
	// location information. (We are guaranteed at least one by the above logic.)
	for name, blocks := range blockAttrVals {
		ret.Attributes[name] = &hcl.Attribute{
			Name: name,
			Expr: &fixupBlocksExpr{
				blocks: blocks,
				ety:    b.schema.Attributes[name].Type.ElementType(),
			},

			Range:     blocks[0].DefRange,
			NameRange: blocks[0].TypeRange,
		}
	}
	return &ret
}

type fixupBlocksExpr struct {
	blocks hcl.Blocks
	ety    cty.Type
}

func (e *fixupBlocksExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	// In order to produce a suitable value for our expression we need to
	// now decode the whole descendent block structure under each of our block
	// bodies.
	//
	// That requires us to do something rather strange: we must construct a
	// synthetic block type schema derived from the element type of the
	// attribute, thus inverting our usual direction of lowering a schema
	// into an implied type. Because a type is less detailed than a schema,
	// the result is imprecise and in particular will just consider all
	// the attributes to be optional and let the provider eventually decide
	// whether to return errors if they turn out to be null when required.
	schema := schemaForCtyType(e.ety) // this schema's ImpliedType will match e.ety
	spec := schema.DecoderSpec()

	vals := make([]cty.Value, len(e.blocks))
	var diags hcl.Diagnostics
	for i, block := range e.blocks {
		val, blockDiags := hcldec.Decode(block.Body, spec, ctx)
		diags = append(diags, blockDiags...)
		if val == cty.NilVal {
			val = cty.UnknownVal(e.ety)
		}
		vals[i] = val
	}
	if len(vals) == 0 {
		return cty.ListValEmpty(e.ety), diags
	}
	return cty.ListVal(vals), diags
}

func (e *fixupBlocksExpr) Variables() []hcl.Traversal {
	var ret []hcl.Traversal
	schema := schemaForCtyType(e.ety)
	spec := schema.DecoderSpec()
	for _, block := range e.blocks {
		ret = append(ret, hcldec.Variables(block.Body, spec)...)
	}
	return ret
}

func (e *fixupBlocksExpr) Range() hcl.Range {
	// This is not really an appropriate range for the expression but it's
	// the best we can do from here.
	return e.blocks[0].DefRange
}

func (e *fixupBlocksExpr) StartRange() hcl.Range {
	return e.blocks[0].DefRange
}

// schemaForCtyType converts a cty object type into an approximately-equivalent
// configschema.Block. If the given type is not an object type then this
// function will panic.
func schemaForCtyType(ty cty.Type) *configschema.Block {
	atys := ty.AttributeTypes()
	ret := &configschema.Block{
		Attributes: make(map[string]*configschema.Attribute, len(atys)),
	}
	for name, aty := range atys {
		ret.Attributes[name] = &configschema.Attribute{
			Type:     aty,
			Optional: true,
		}
	}
	return ret
}
