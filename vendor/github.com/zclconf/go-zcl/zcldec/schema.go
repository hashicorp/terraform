package zcldec

import (
	"github.com/zclconf/go-zcl/zcl"
)

// ImpliedSchema returns the *zcl.BodySchema implied by the given specification.
// This is the schema that the Decode function will use internally to
// access the content of a given body.
func ImpliedSchema(spec Spec) *zcl.BodySchema {
	var attrs []zcl.AttributeSchema
	var blocks []zcl.BlockHeaderSchema

	// visitSameBodyChildren walks through the spec structure, calling
	// the given callback for each descendent spec encountered. We are
	// interested in the specs that reference attributes and blocks.
	visit := func(s Spec) {
		if as, ok := s.(attrSpec); ok {
			attrs = append(attrs, as.attrSchemata()...)
		}

		if bs, ok := s.(blockSpec); ok {
			blocks = append(blocks, bs.blockHeaderSchemata()...)
		}
	}

	visit(spec)
	spec.visitSameBodyChildren(visit)

	return &zcl.BodySchema{
		Attributes: attrs,
		Blocks:     blocks,
	}
}
