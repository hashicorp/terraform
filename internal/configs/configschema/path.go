// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"github.com/zclconf/go-cty/cty"
)

// AttributeByPath looks up the Attribute schema which corresponds to the given
// cty.Path. A nil value is returned if the given path does not correspond to a
// specific attribute.
func (b *Block) AttributeByPath(path cty.Path) *Attribute {
	block := b
	for i, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if attr := block.Attributes[step.Name]; attr != nil {
				// If the Attribute is defined with a NestedType and there's
				// more to the path, descend into the NestedType
				if attr.NestedType != nil && i < len(path)-1 {
					return attr.NestedType.AttributeByPath(path[i+1:])
				} else if i < len(path)-1 { // There's more to the path, but not more to this Attribute.
					return nil
				}
				return attr
			}

			if nestedBlock := block.BlockTypes[step.Name]; nestedBlock != nil {
				block = &nestedBlock.Block
				continue
			}

			return nil
		}
	}
	return nil
}

// AttributeByPath recurses through a NestedType to look up the Attribute scheme
// which corresponds to the given cty.Path. A nil value is returned if the given
// path does not correspond to a specific attribute.
func (o *Object) AttributeByPath(path cty.Path) *Attribute {
	for i, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if attr := o.Attributes[step.Name]; attr != nil {
				if attr.NestedType != nil && i < len(path)-1 {
					return attr.NestedType.AttributeByPath(path[i+1:])
				} else if i < len(path)-1 { // There's more to the path, but not more to this Attribute.
					return nil
				}
				return attr
			}
		}
	}
	return nil
}
