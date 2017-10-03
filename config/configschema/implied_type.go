package configschema

import (
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

	attrTypes := map[string]cty.Type{}

	for name, attrS := range b.Attributes {
		attrTypes[name] = attrS.Type
	}

	for name, blockS := range b.BlockTypes {
		if _, exists := attrTypes[name]; exists {
			// This indicates an invalid schema, since it's not valid to
			// define both an attribute and a block type of the same name.
			// However, we don't raise this here since it's checked by
			// InternalValidate.
			continue
		}

		childType := blockS.Block.ImpliedType()
		switch blockS.Nesting {
		case NestingSingle:
			attrTypes[name] = childType
		case NestingList:
			attrTypes[name] = cty.List(childType)
		case NestingSet:
			attrTypes[name] = cty.Set(childType)
		case NestingMap:
			attrTypes[name] = cty.Map(childType)
		default:
			// Invalid nesting type is just ignored. It's checked by
			// InternalValidate.
			continue
		}
	}

	return cty.Object(attrTypes)
}
