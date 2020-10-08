package blocktoattr

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func ambiguousNames(schema *configschema.Block) map[string]struct{} {
	if schema == nil {
		return nil
	}
	ambiguousNames := make(map[string]struct{})
	for name, attrS := range schema.Attributes {
		aty := attrS.Type
		if (aty.IsListType() || aty.IsSetType()) && aty.ElementType().IsObjectType() {
			ambiguousNames[name] = struct{}{}
		}
	}
	return ambiguousNames
}

func effectiveSchema(given *hcl.BodySchema, body hcl.Body, ambiguousNames map[string]struct{}, dynamicExpanded bool) *hcl.BodySchema {
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

		for name := range ambiguousNames {
			probeSchema = hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: name,
					},
				},
			}
			content, _, _ := body.PartialContent(&probeSchema)
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
			content, _, _ = body.PartialContent(&probeSchema)
			if len(content.Blocks) > 0 || dynamicExpanded {
				// A dynamic block with an empty iterator returns nothing.
				// If there's no attribute and we have either a block or a
				// dynamic expansion, we need to rewrite this one as a
				// block for a successful result.
				appearsAsBlock[name] = struct{}{}
			}
		}
		if !dynamicExpanded {
			// If we're deciding for a context where dynamic blocks haven't
			// been expanded yet then we need to probe for those too.
			probeSchema = hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type:       "dynamic",
						LabelNames: []string{"type"},
					},
				},
			}
			content, _, _ := body.PartialContent(&probeSchema)
			for _, block := range content.Blocks {
				if _, exists := ambiguousNames[block.Labels[0]]; exists {
					appearsAsBlock[block.Labels[0]] = struct{}{}
				}
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

// SchemaForCtyElementType converts a cty object type into an
// approximately-equivalent configschema.Block representing the element of
// a list or set. If the given type is not an object type then this
// function will panic.
func SchemaForCtyElementType(ty cty.Type) *configschema.Block {
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

// SchemaForCtyContainerType converts a cty list-of-object or set-of-object type
// into an approximately-equivalent configschema.NestedBlock. If the given type
// is not of the expected kind then this function will panic.
func SchemaForCtyContainerType(ty cty.Type) *configschema.NestedBlock {
	var nesting configschema.NestingMode
	switch {
	case ty.IsListType():
		nesting = configschema.NestingList
	case ty.IsSetType():
		nesting = configschema.NestingSet
	default:
		panic("unsuitable type")
	}
	nested := SchemaForCtyElementType(ty.ElementType())
	return &configschema.NestedBlock{
		Nesting: nesting,
		Block:   *nested,
	}
}

// TypeCanBeBlocks returns true if the given type is a list-of-object or
// set-of-object type, and would thus be subject to the blocktoattr fixup
// if used as an attribute type.
func TypeCanBeBlocks(ty cty.Type) bool {
	return (ty.IsListType() || ty.IsSetType()) && ty.ElementType().IsObjectType()
}
