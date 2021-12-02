package configschema

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

// ValueMarks returns a set of path value marks for a given value and path,
// based on the sensitive flag for each attribute within the schema. Nested
// blocks are descended (if present in the given value).
func (b *Block) ValueMarks(val cty.Value, path cty.Path) []cty.PathValueMarks {
	var pvm []cty.PathValueMarks

	// We can mark attributes as sensitive even if the value is null
	for name, attrS := range b.Attributes {
		if attrS.Sensitive {
			// Create a copy of the path, with this step added, to add to our PathValueMarks slice
			attrPath := make(cty.Path, len(path), len(path)+1)
			copy(attrPath, path)
			attrPath = append(path, cty.GetAttrStep{Name: name})
			pvm = append(pvm, cty.PathValueMarks{
				Path:  attrPath,
				Marks: cty.NewValueMarks(marks.Sensitive),
			})
		}
	}

	// If the value is null, no other marks are possible
	if val.IsNull() {
		return pvm
	}

	// Extract marks for nested attribute type values
	for name, attrS := range b.Attributes {
		// If the attribute has no nested type, or the nested type doesn't
		// contain any sensitive attributes, skip inspecting it
		if attrS.NestedType == nil || !attrS.NestedType.ContainsSensitive() {
			continue
		}

		// Create a copy of the path, with this step added, to add to our PathValueMarks slice
		attrPath := make(cty.Path, len(path), len(path)+1)
		copy(attrPath, path)
		attrPath = append(path, cty.GetAttrStep{Name: name})

		pvm = append(pvm, attrS.NestedType.ValueMarks(val.GetAttr(name), attrPath)...)
	}

	// Extract marks for nested blocks
	for name, blockS := range b.BlockTypes {
		// If our block doesn't contain any sensitive attributes, skip inspecting it
		if !blockS.Block.ContainsSensitive() {
			continue
		}

		blockV := val.GetAttr(name)
		if blockV.IsNull() || !blockV.IsKnown() {
			continue
		}

		// Create a copy of the path, with this step added, to add to our PathValueMarks slice
		blockPath := make(cty.Path, len(path), len(path)+1)
		copy(blockPath, path)
		blockPath = append(path, cty.GetAttrStep{Name: name})

		switch blockS.Nesting {
		case NestingSingle, NestingGroup:
			pvm = append(pvm, blockS.Block.ValueMarks(blockV, blockPath)...)
		case NestingList, NestingMap, NestingSet:
			for it := blockV.ElementIterator(); it.Next(); {
				idx, blockEV := it.Element()
				morePaths := blockS.Block.ValueMarks(blockEV, append(blockPath, cty.IndexStep{Key: idx}))
				pvm = append(pvm, morePaths...)
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}
	}
	return pvm
}

// ValueMarks returns a set of path value marks for a given value and path,
// based on the sensitive flag for each attribute within the nested attribute.
// Attributes with nested types are descended (if present in the given value).
func (o *Object) ValueMarks(val cty.Value, path cty.Path) []cty.PathValueMarks {
	var pvm []cty.PathValueMarks

	if val.IsNull() || !val.IsKnown() {
		return pvm
	}

	for name, attrS := range o.Attributes {
		// Skip attributes which can never produce sensitive path value marks
		if !attrS.Sensitive && (attrS.NestedType == nil || !attrS.NestedType.ContainsSensitive()) {
			continue
		}

		switch o.Nesting {
		case NestingSingle, NestingGroup:
			// Create a path to this attribute
			attrPath := make(cty.Path, len(path), len(path)+1)
			copy(attrPath, path)
			attrPath = append(path, cty.GetAttrStep{Name: name})

			if attrS.Sensitive {
				// If the entire attribute is sensitive, mark it so
				pvm = append(pvm, cty.PathValueMarks{
					Path:  attrPath,
					Marks: cty.NewValueMarks(marks.Sensitive),
				})
			} else {
				// The attribute has a nested type which contains sensitive
				// attributes, so recurse
				pvm = append(pvm, attrS.NestedType.ValueMarks(val.GetAttr(name), attrPath)...)
			}
		case NestingList, NestingMap, NestingSet:
			// For nested attribute types which have a non-single nesting mode,
			// we add path value marks for each element of the collection
			for it := val.ElementIterator(); it.Next(); {
				idx, attrEV := it.Element()
				attrV := attrEV.GetAttr(name)

				// Create a path to this element of the attribute's collection. Note
				// that the path is extended in opposite order to the iteration order
				// of the loops: index into the collection, then the contained
				// attribute name. This is because we have one type
				// representing multiple collection elements.
				attrPath := make(cty.Path, len(path), len(path)+2)
				copy(attrPath, path)
				attrPath = append(path, cty.IndexStep{Key: idx}, cty.GetAttrStep{Name: name})

				if attrS.Sensitive {
					// If the entire attribute is sensitive, mark it so
					pvm = append(pvm, cty.PathValueMarks{
						Path:  attrPath,
						Marks: cty.NewValueMarks(marks.Sensitive),
					})
				} else {
					// The attribute has a nested type which contains sensitive
					// attributes, so recurse
					pvm = append(pvm, attrS.NestedType.ValueMarks(attrV, attrPath)...)
				}
			}
		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", attrS.NestedType.Nesting))
		}
	}
	return pvm
}
