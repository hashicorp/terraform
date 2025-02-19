// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"fmt"
	"slices"

	"github.com/zclconf/go-cty/cty"
)

// WARNING: WriteOnlyPaths must exactly mirror the SensitivePaths method, since
// they both use the same process just for different attribute types. Any fixes
// here must be made in SensitivePaths, and vice versa.

// WriteOnlyPaths returns a set of paths into the given value that should
// be marked as write-only based on the static declarations in the schema.
func (b *Block) WriteOnlyPaths(val cty.Value, basePath cty.Path) []cty.Path {
	var ret []cty.Path

	// a block cannot be write-only, so nothing to return
	if val.IsNull() || !val.IsKnown() {
		return ret
	}

	for name, attrS := range b.Attributes {
		if attrS.WriteOnly {
			attrPath := slices.Concat(basePath, cty.GetAttrPath(name))
			ret = append(ret, attrPath)
		}
	}

	// Extract paths for marks from nested attribute type values
	for name, attrS := range b.Attributes {
		// If the attribute has no nested type, or the nested type doesn't
		// contain any write-only attributes, skip inspecting it
		if attrS.NestedType == nil || !attrS.NestedType.ContainsWriteOnly() {
			continue
		}

		// Create a copy of the path, with this step added, to add to our PathValueMarks slice
		attrPath := slices.Concat(basePath, cty.GetAttrPath(name))
		ret = append(ret, attrS.NestedType.writeOnlyPaths(val.GetAttr(name), attrPath)...)
	}

	// Extract paths for marks from nested blocks
	for name, blockS := range b.BlockTypes {
		// If our block doesn't contain any write-only attributes, skip inspecting it
		if !blockS.Block.ContainsWriteOnly() {
			continue
		}

		blockV := val.GetAttr(name)
		if blockV.IsNull() || !blockV.IsKnown() {
			continue
		}

		// Create a copy of the path, with this step added, to add to our PathValueMarks slice
		blockPath := slices.Concat(basePath, cty.GetAttrPath(name))

		switch blockS.Nesting {
		case NestingSingle, NestingGroup:
			ret = append(ret, blockS.Block.WriteOnlyPaths(blockV, blockPath)...)
		case NestingList, NestingMap, NestingSet:
			blockV, _ = blockV.Unmark() // peel off one level of marking so we can iterate
			for it := blockV.ElementIterator(); it.Next(); {
				idx, blockEV := it.Element()
				// Create a copy of the path, with this block instance's index
				// step added, to add to our PathValueMarks slice
				blockInstancePath := slices.Concat(blockPath, cty.IndexPath(idx))
				morePaths := blockS.Block.WriteOnlyPaths(blockEV, blockInstancePath)
				ret = append(ret, morePaths...)
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}
	}
	return ret
}

// writeOnlyPaths returns a set of paths into the given value that should be
// marked as write-only based on the static declarations in the schema.
func (o *Object) writeOnlyPaths(val cty.Value, basePath cty.Path) []cty.Path {
	var ret []cty.Path

	if val.IsNull() || !val.IsKnown() {
		return ret
	}

	for name, attrS := range o.Attributes {
		// Skip attributes which can never produce write-only path value marks
		if !attrS.WriteOnly && (attrS.NestedType == nil || !attrS.NestedType.ContainsWriteOnly()) {
			continue
		}

		switch o.Nesting {
		case NestingSingle, NestingGroup:
			// Create a path to this attribute
			attrPath := slices.Concat(basePath, cty.GetAttrPath(name))
			if attrS.WriteOnly {
				// If the entire attribute is write-only, mark it so
				ret = append(ret, attrPath)
			} else {
				// The attribute has a nested type which contains write-only
				// attributes, so recurse
				ret = append(ret, attrS.NestedType.writeOnlyPaths(val.GetAttr(name), attrPath)...)
			}
		case NestingList, NestingMap, NestingSet:
			// For nested attribute types which have a non-single nesting mode,
			// we add path value marks for each element of the collection
			val, _ = val.Unmark() // peel off one level of marking so we can iterate
			for it := val.ElementIterator(); it.Next(); {
				idx, attrEV := it.Element()
				attrV := attrEV.GetAttr(name)

				// Create a path to this element of the attribute's collection. Note
				// that the path is extended in opposite order to the iteration order
				// of the loops: index into the collection, then the contained
				// attribute name. This is because we have one type
				// representing multiple collection elements.
				attrPath := slices.Concat(basePath, cty.IndexPath(idx).GetAttr(name))

				if attrS.WriteOnly {
					// If the entire attribute is write-only, mark it so
					ret = append(ret, attrPath)
				} else {
					ret = append(ret, attrS.NestedType.writeOnlyPaths(attrV, attrPath)...)
				}
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", o.Nesting))
		}
	}
	return ret
}
