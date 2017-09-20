package hcldec

import (
	"github.com/hashicorp/hcl2/hcl"
)

// ImpliedSchema returns the *hcl.BodySchema implied by the given specification.
// This is the schema that the Decode function will use internally to
// access the content of a given body.
func ImpliedSchema(spec Spec) *hcl.BodySchema {
	var attrs []hcl.AttributeSchema
	var blocks []hcl.BlockHeaderSchema

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

	return &hcl.BodySchema{
		Attributes: attrs,
		Blocks:     blocks,
	}
}
