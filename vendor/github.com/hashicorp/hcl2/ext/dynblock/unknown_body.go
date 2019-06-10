package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// unknownBody is a funny body that just reports everything inside it as
// unknown. It uses a given other body as a sort of template for what attributes
// and blocks are inside -- including source location information -- but
// subsitutes unknown values of unknown type for all attributes.
//
// This rather odd process is used to handle expansion of dynamic blocks whose
// for_each expression is unknown. Since a block cannot itself be unknown,
// we instead arrange for everything _inside_ the block to be unknown instead,
// to give the best possible approximation.
type unknownBody struct {
	template hcl.Body
}

var _ hcl.Body = unknownBody{}

func (b unknownBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	content, diags := b.template.Content(schema)
	content = b.fixupContent(content)

	// We're intentionally preserving the diagnostics reported from the
	// inner body so that we can still report where the template body doesn't
	// match the requested schema.
	return content, diags
}

func (b unknownBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	content, remain, diags := b.template.PartialContent(schema)
	content = b.fixupContent(content)
	remain = unknownBody{remain} // remaining content must also be wrapped

	// We're intentionally preserving the diagnostics reported from the
	// inner body so that we can still report where the template body doesn't
	// match the requested schema.
	return content, remain, diags
}

func (b unknownBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	attrs, diags := b.template.JustAttributes()
	attrs = b.fixupAttrs(attrs)

	// We're intentionally preserving the diagnostics reported from the
	// inner body so that we can still report where the template body doesn't
	// match the requested schema.
	return attrs, diags
}

func (b unknownBody) MissingItemRange() hcl.Range {
	return b.template.MissingItemRange()
}

func (b unknownBody) fixupContent(got *hcl.BodyContent) *hcl.BodyContent {
	ret := &hcl.BodyContent{}
	ret.Attributes = b.fixupAttrs(got.Attributes)
	if len(got.Blocks) > 0 {
		ret.Blocks = make(hcl.Blocks, 0, len(got.Blocks))
		for _, gotBlock := range got.Blocks {
			new := *gotBlock                      // shallow copy
			new.Body = unknownBody{gotBlock.Body} // nested content must also be marked unknown
			ret.Blocks = append(ret.Blocks, &new)
		}
	}

	return ret
}

func (b unknownBody) fixupAttrs(got hcl.Attributes) hcl.Attributes {
	if len(got) == 0 {
		return nil
	}
	ret := make(hcl.Attributes, len(got))
	for name, gotAttr := range got {
		new := *gotAttr // shallow copy
		new.Expr = hcl.StaticExpr(cty.DynamicVal, gotAttr.Expr.Range())
		ret[name] = &new
	}
	return ret
}
