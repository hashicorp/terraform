package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// ValueMarks returns a set of path value marks for a given value and path,
// based on the sensitive flag for each attribute within the schema. Nested
// blocks are descended (if present in the given value).
func (b *Block) ValueMarks(val cty.Value, path cty.Path) []cty.PathValueMarks {
	var pvm []cty.PathValueMarks
	for name, attrS := range b.Attributes {
		if attrS.Sensitive {
			// Create a copy of the path, with this step added, to add to our PathValueMarks slice
			attrPath := make(cty.Path, len(path), len(path)+1)
			copy(attrPath, path)
			attrPath = append(path, cty.GetAttrStep{Name: name})
			pvm = append(pvm, cty.PathValueMarks{
				Path:  attrPath,
				Marks: cty.NewValueMarks("sensitive"),
			})
		}
	}

	if val.IsNull() {
		return pvm
	}
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
