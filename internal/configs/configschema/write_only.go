// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// WriteOnlyPaths returns a set of paths into the given value that
// are considered write only based on the declarations in the schema.
func (b *Block) WriteOnlyPaths(val cty.Value, basePath cty.Path) []cty.Path {
	var ret []cty.Path

	for name, attrS := range b.Attributes {
		if attrS.WriteOnly {
			attrPath := copyAndExtendPath(basePath, cty.GetAttrStep{Name: name})
			ret = append(ret, attrPath)
		}

		if attrS.NestedType == nil || !attrS.NestedType.ContainsWriteOnly() {
			continue
		}

		attrPath := copyAndExtendPath(basePath, cty.GetAttrStep{Name: name})
		ret = append(ret, attrS.NestedType.WriteOnlyPaths(attrPath)...)
	}

	// Extract from nested blocks
	for name, blockS := range b.BlockTypes {
		// If our block doesn't contain any write only attributes, skip inspecting it
		if !blockS.Block.ContainsWriteOnly() {
			continue
		}

		if val.IsNull() {
			return ret
		}

		blockV := val.GetAttr(name)

		// Create a copy of the path, with this step added, to add to our PathValueMarks slice
		blockPath := copyAndExtendPath(basePath, cty.GetAttrStep{Name: name})

		switch blockS.Nesting {
		case NestingSingle, NestingGroup:
			ret = append(ret, blockS.Block.WriteOnlyPaths(blockV, blockPath)...)
		case NestingList, NestingMap, NestingSet:
			blockV, _ = blockV.Unmark() // peel off one level of marking so we can iterate

			if blockV.IsNull() || !blockV.IsKnown() {
				return ret
			}

			for it := blockV.ElementIterator(); it.Next(); {
				idx, blockEV := it.Element()

				blockInstancePath := copyAndExtendPath(blockPath, cty.IndexStep{Key: idx})
				morePaths := blockS.Block.WriteOnlyPaths(blockEV, blockInstancePath)
				ret = append(ret, morePaths...)
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}
	}
	return ret
}

// WriteOnlyPaths returns a set of paths into the given value that
// are considered write only based on the declarations in the schema.
func (o *Object) WriteOnlyPaths(basePath cty.Path) []cty.Path {
	var ret []cty.Path

	for name, attrS := range o.Attributes {
		// Create a path to this attribute
		attrPath := copyAndExtendPath(basePath, cty.GetAttrStep{Name: name})

		switch o.Nesting {
		case NestingSingle, NestingGroup:
			if attrS.WriteOnly {
				ret = append(ret, attrPath)
			} else {
				// The attribute has a nested type which contains write only
				// attributes, so recurse
				ret = append(ret, attrS.NestedType.WriteOnlyPaths(attrPath)...)
			}
		case NestingList, NestingMap, NestingSet:
			// If the attribute is iterable type we assume that
			// it is write only in its entirety since we cannot
			// construct indexes from a null value.
			if attrS.WriteOnly {
				ret = append(ret, attrPath)
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", attrS.NestedType.Nesting))
		}
	}
	return ret
}
