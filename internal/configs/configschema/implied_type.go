// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

// ImpliedType returns the cty.Type that would result from decoding a
// configuration block using the receiving block schema.
//
// The type returned from Block.ImpliedType differs from the type returned by
// hcldec.ImpliedType in that there will be no objects with optional
// attributes, since this value is not to be used for the decoding of
// configuration.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Block objects should be
// tested using the InternalValidate method to detect any inconsistencies
// that would cause this method to fall back on defaults and assumptions.
func (b *Block) ImpliedType() cty.Type {
	return b.specType().WithoutOptionalAttributesDeep()
}

// specType returns the cty.Type used for decoding a configuration
// block using the receiving block schema. This is the type used internally by
// hcldec to decode configuration.
func (b *Block) specType() cty.Type {
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
		if attrS.NestedType != nil && attrS.NestedType.ContainsSensitive() {
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

// ImpliedType returns the cty.Type that would result from decoding a Block's
// ImpliedType and getting the resulting AttributeType.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Object objects should be tested
// using the InternalValidate method to detect any inconsistencies that would
// cause this method to fall back on defaults and assumptions.
func (a *Attribute) ImpliedType() cty.Type {
	if a.NestedType != nil {
		return a.NestedType.specType().WithoutOptionalAttributesDeep()
	}
	return a.Type
}

// ImpliedType returns the cty.Type that would result from decoding a
// NestedType Attribute using the receiving block schema.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Object objects should be tested
// using the InternalValidate method to detect any inconsistencies that would
// cause this method to fall back on defaults and assumptions.
func (o *Object) ImpliedType() cty.Type {
	return o.specType().WithoutOptionalAttributesDeep()
}

// specType returns the cty.Type used for decoding a NestedType Attribute using
// the receiving block schema.
func (o *Object) specType() cty.Type {
	if o == nil {
		return cty.EmptyObject
	}

	attrTys := make(map[string]cty.Type, len(o.Attributes))
	for name, attrS := range o.Attributes {
		if attrS.NestedType != nil {
			attrTys[name] = attrS.NestedType.specType()
		} else {
			attrTys[name] = attrS.Type
		}
	}
	optAttrs := listOptionalAttrsFromObject(o)

	var ret cty.Type
	if len(optAttrs) > 0 {
		ret = cty.ObjectWithOptionalAttrs(attrTys, optAttrs)
	} else {
		ret = cty.Object(attrTys)
	}
	switch o.Nesting {
	case NestingSingle:
		return ret
	case NestingList:
		return cty.List(ret)
	case NestingMap:
		return cty.Map(ret)
	case NestingSet:
		return cty.Set(ret)
	default: // Should never happen
		return cty.EmptyObject
	}
}

// ContainsSensitive returns true if any of the attributes of the receiving
// Object are marked as sensitive.
func (o *Object) ContainsSensitive() bool {
	for _, attrS := range o.Attributes {
		if attrS.Sensitive {
			return true
		}
		if attrS.NestedType != nil && attrS.NestedType.ContainsSensitive() {
			return true
		}
	}
	return false
}
