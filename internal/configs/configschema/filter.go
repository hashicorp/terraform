// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import "github.com/zclconf/go-cty/cty"

type FilterT[T any] func(cty.Path, T) bool

var (
	FilterReadOnlyAttribute = func(path cty.Path, attribute *Attribute) bool {
		return attribute.Computed && !attribute.Optional
	}

	FilterHelperSchemaIdAttribute = func(path cty.Path, attribute *Attribute) bool {
		if path.Equals(cty.GetAttrPath("id")) && attribute.Computed && attribute.Optional {
			return true
		}
		return false
	}

	FilterDeprecatedAttribute = func(path cty.Path, attribute *Attribute) bool {
		return attribute.Deprecated
	}

	FilterDeprecatedBlock = func(path cty.Path, block *NestedBlock) bool {
		return block.Deprecated
	}
)

func FilterOr[T any](filters ...FilterT[T]) FilterT[T] {
	return func(path cty.Path, value T) bool {
		for _, f := range filters {
			if f(path, value) {
				return true
			}
		}
		return false
	}
}

func (b *Block) Filter(filterAttribute FilterT[*Attribute], filterBlock FilterT[*NestedBlock]) *Block {
	return b.filter(nil, filterAttribute, filterBlock)
}

func (b *Block) filter(path cty.Path, filterAttribute FilterT[*Attribute], filterBlock FilterT[*NestedBlock]) *Block {
	ret := &Block{
		Description:     b.Description,
		DescriptionKind: b.DescriptionKind,
		Deprecated:      b.Deprecated,
	}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, attrS := range b.Attributes {
		path := path.GetAttr(name)
		if filterAttribute == nil || !filterAttribute(path, attrS) {
			attr := *attrS
			if attrS.NestedType != nil {
				attr.NestedType = filterNestedType(attrS.NestedType, path, filterAttribute)
			}
			ret.Attributes[name] = &attr
		}
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		path := path.GetAttr(name)
		if filterBlock == nil || !filterBlock(path, blockS) {
			block := blockS.filter(path, filterAttribute, filterBlock)
			ret.BlockTypes[name] = &NestedBlock{
				Block:    *block,
				Nesting:  blockS.Nesting,
				MinItems: blockS.MinItems,
				MaxItems: blockS.MaxItems,
			}
		}
	}

	return ret
}

func filterNestedType(obj *Object, path cty.Path, filterAttribute FilterT[*Attribute]) *Object {
	if obj == nil {
		return nil
	}

	ret := &Object{
		Attributes: map[string]*Attribute{},
		Nesting:    obj.Nesting,
	}

	for name, attrS := range obj.Attributes {
		path := path.GetAttr(name)
		if filterAttribute == nil || !filterAttribute(path, attrS) {
			attr := *attrS
			if attrS.NestedType != nil {
				attr.NestedType = filterNestedType(attrS.NestedType, path, filterAttribute)
			}
			ret.Attributes[name] = &attr
		}
	}

	return ret
}
