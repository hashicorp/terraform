package plugin

import (
	"fmt"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// SetUnknowns takes a cty.Value, and compares it to the schema setting any null
// values which are computed to unknown.
func SetUnknowns(val cty.Value, schema *configschema.Block) cty.Value {
	if !val.IsKnown() {
		return val
	}

	// If the object was null, we still need to handle the top level attributes
	// which might be computed, but we don't need to expand the blocks.
	if val.IsNull() {
		objMap := map[string]cty.Value{}
		allNull := true
		for name, attr := range schema.Attributes {
			switch {
			case attr.Computed:
				objMap[name] = cty.UnknownVal(attr.Type)
				allNull = false
			default:
				objMap[name] = cty.NullVal(attr.Type)
			}
		}

		// If this object has no unknown attributes, then we can leave it null.
		if allNull {
			return val
		}

		return cty.ObjectVal(objMap)
	}

	valMap := val.AsValueMap()
	newVals := make(map[string]cty.Value)

	for name, attr := range schema.Attributes {
		v := valMap[name]

		if attr.Computed && v.IsNull() {
			newVals[name] = cty.UnknownVal(attr.Type)
			continue
		}

		newVals[name] = v
	}

	for name, blockS := range schema.BlockTypes {
		blockVal := valMap[name]
		if blockVal.IsNull() || !blockVal.IsKnown() {
			newVals[name] = blockVal
			continue
		}

		blockValType := blockVal.Type()
		blockElementType := blockS.Block.ImpliedType()

		// This switches on the value type here, so we can correctly switch
		// between Tuples/Lists and Maps/Objects.
		switch {
		case blockS.Nesting == configschema.NestingSingle:
			// NestingSingle is the only exception here, where we treat the
			// block directly as an object
			newVals[name] = SetUnknowns(blockVal, &blockS.Block)

		case blockValType.IsSetType(), blockValType.IsListType(), blockValType.IsTupleType():
			listVals := blockVal.AsValueSlice()
			newListVals := make([]cty.Value, 0, len(listVals))

			for _, v := range listVals {
				newListVals = append(newListVals, SetUnknowns(v, &blockS.Block))
			}

			switch {
			case blockValType.IsSetType():
				switch len(newListVals) {
				case 0:
					newVals[name] = cty.SetValEmpty(blockElementType)
				default:
					newVals[name] = cty.SetVal(newListVals)
				}
			case blockValType.IsListType():
				switch len(newListVals) {
				case 0:
					newVals[name] = cty.ListValEmpty(blockElementType)
				default:
					newVals[name] = cty.ListVal(newListVals)
				}
			case blockValType.IsTupleType():
				newVals[name] = cty.TupleVal(newListVals)
			}

		case blockValType.IsMapType(), blockValType.IsObjectType():
			mapVals := blockVal.AsValueMap()
			newMapVals := make(map[string]cty.Value)

			for k, v := range mapVals {
				newMapVals[k] = SetUnknowns(v, &blockS.Block)
			}

			switch {
			case blockValType.IsMapType():
				switch len(newMapVals) {
				case 0:
					newVals[name] = cty.MapValEmpty(blockElementType)
				default:
					newVals[name] = cty.MapVal(newMapVals)
				}
			case blockValType.IsObjectType():
				if len(newMapVals) == 0 {
					// We need to populate empty values to make a valid object.
					for attr, ty := range blockElementType.AttributeTypes() {
						newMapVals[attr] = cty.NullVal(ty)
					}
				}
				newVals[name] = cty.ObjectVal(newMapVals)
			}

		default:
			panic(fmt.Sprintf("failed to set unknown values for nested block %q:%#v", name, blockValType))
		}
	}

	return cty.ObjectVal(newVals)
}
