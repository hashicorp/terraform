package configschema

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

// ImpliedType returns the cty.Type that would result from decoding a
// configuration block using the receiving block schema.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Block objects should be
// tested using the InternalValidate method to detect any inconsistencies
// that would cause this method to fall back on defaults and assumptions.
func (b *Block) ImpliedType() cty.Type {
	if b == nil {
		return cty.EmptyObject
	}

	return hcldec.ImpliedType(b.DecoderSpec())
}

// ContainsSensitive returns true if any of the attributes of the receiving
// block or any of its descendent blocks are marked as sensitive.
//
// Blocks themselves cannot be sensitive as a whole -- sensitivity is a
// per-attribute idea -- but sometimes we want to include a whole object
// decoded from a block in some UI output, and that is safe to do only if
// none of the contained attributes are sensitive.
func (b *Block) ContainsSensitive() bool {
	for _, attrS := range b.Attributes {
		if attrS.Sensitive {
			return true
		}
	}
	for _, blockS := range b.BlockTypes {
		if blockS.ContainsSensitive() {
			return true
		}
	}
	return false
}

// ImpliedType returns the cty.Type that would result from decoding a NestedType
// Attribute using the receiving block schema.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Object objects should be tested
// using the InternalValidate method to detect any inconsistencies that would
// cause this method to fall back on defaults and assumptions.
func (o *Object) ImpliedType() cty.Type {
	if o == nil {
		return cty.EmptyObject
	}

	attrTys := make(map[string]cty.Type, len(o.Attributes))
	for name, attrS := range o.Attributes {
		attrTys[name] = attrS.Type
	}

	var optAttrs []string
	optAttrs = listOptionalAttrsFromObject(o, optAttrs)

	return cty.ObjectWithOptionalAttrs(attrTys, optAttrs)
}

// ContainsSensitive returns true if any of the attributes of the receiving
// Object are marked as sensitive.
func (o *Object) ContainsSensitive() bool {
	for _, attrS := range o.Attributes {
		if attrS.Sensitive {
			return true
		}
	}
	return false
}
