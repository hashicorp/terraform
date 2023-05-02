// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"github.com/zclconf/go-cty/cty"
)

// EmptyValue returns the "empty value" for the recieving block, which for
// a block type is a non-null object where all of the attribute values are
// the empty values of the block's attributes and nested block types.
//
// In other words, it returns the value that would be returned if an empty
// block were decoded against the recieving schema, assuming that no required
// attribute or block constraints were honored.
func (b *Block) EmptyValue() cty.Value {
	vals := make(map[string]cty.Value)
	for name, attrS := range b.Attributes {
		vals[name] = attrS.EmptyValue()
	}
	for name, blockS := range b.BlockTypes {
		vals[name] = blockS.EmptyValue()
	}
	return cty.ObjectVal(vals)
}

// EmptyValue returns the "empty value" for the receiving attribute, which is
// the value that would be returned if there were no definition of the attribute
// at all, ignoring any required constraint.
func (a *Attribute) EmptyValue() cty.Value {
	return cty.NullVal(a.ImpliedType())
}

// EmptyValue returns the "empty value" for when there are zero nested blocks
// present of the receiving type.
func (b *NestedBlock) EmptyValue() cty.Value {
	switch b.Nesting {
	case NestingSingle:
		return cty.NullVal(b.Block.ImpliedType())
	case NestingGroup:
		return b.Block.EmptyValue()
	case NestingList:
		if ty := b.Block.ImpliedType(); ty.HasDynamicTypes() {
			return cty.EmptyTupleVal
		} else {
			return cty.ListValEmpty(ty)
		}
	case NestingMap:
		if ty := b.Block.ImpliedType(); ty.HasDynamicTypes() {
			return cty.EmptyObjectVal
		} else {
			return cty.MapValEmpty(ty)
		}
	case NestingSet:
		return cty.SetValEmpty(b.Block.ImpliedType())
	default:
		// Should never get here because the above is intended to be exhaustive,
		// but we'll be robust and return a result nonetheless.
		return cty.NullVal(cty.DynamicPseudoType)
	}
}
