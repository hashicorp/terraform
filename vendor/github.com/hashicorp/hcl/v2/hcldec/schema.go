package hcldec

import (
	"github.com/hashicorp/hcl/v2"
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
	var visit visitFunc
	visit = func(s Spec) {
		if as, ok := s.(attrSpec); ok {
			attrs = append(attrs, as.attrSchemata()...)
		}

		if bs, ok := s.(blockSpec); ok {
			blocks = append(blocks, bs.blockHeaderSchemata()...)
		}

		s.visitSameBodyChildren(visit)
	}

	visit(spec)

	return &hcl.BodySchema{
		Attributes: attrs,
		Blocks:     blocks,
	}
}
