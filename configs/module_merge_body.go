package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

func mergeBodies(base, override hcl.Body) hcl.Body {
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
	panic("mergeBody.Content not yet implemented")
}

func (b mergeBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	panic("mergeBody.Content not yet implemented")
}

func (b mergeBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	panic("mergeBody.JustAttributes not yet implemented")
}

func (b mergeBody) MissingItemRange() hcl.Range {
	return b.Base.MissingItemRange()
}
