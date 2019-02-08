package plugin

import (
	"fmt"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// SetUnknowns takes a cty.Value, and compares it to the schema setting any null
// leaf values which are computed as unknown.
func SetUnknowns(val cty.Value, schema *configschema.Block) cty.Value {
	if val.IsNull() || !val.IsKnown() {
		return val
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

		blockType := blockS.Block.ImpliedType()

		switch blockS.Nesting {
		case configschema.NestingSingle:
			newVals[name] = SetUnknowns(blockVal, &blockS.Block)
		case configschema.NestingSet, configschema.NestingList:
			listVals := blockVal.AsValueSlice()
			newListVals := make([]cty.Value, 0, len(listVals))

			for _, v := range listVals {
				newListVals = append(newListVals, SetUnknowns(v, &blockS.Block))
			}

			switch blockS.Nesting {
			case configschema.NestingSet:
				switch len(newListVals) {
				case 0:
					newVals[name] = cty.SetValEmpty(blockType)
				default:
					newVals[name] = cty.SetVal(newListVals)
				}
			case configschema.NestingList:
				switch len(newListVals) {
				case 0:
					newVals[name] = cty.ListValEmpty(blockType)
				default:
					newVals[name] = cty.ListVal(newListVals)
				}
			}

		case configschema.NestingMap:
			mapVals := blockVal.AsValueMap()
			newMapVals := make(map[string]cty.Value)

			for k, v := range mapVals {
				newMapVals[k] = SetUnknowns(v, &blockS.Block)
			}

			switch len(newMapVals) {
			case 0:
				newVals[name] = cty.MapValEmpty(blockType)
			default:
				newVals[name] = cty.MapVal(newMapVals)
			}

		default:
			panic(fmt.Sprintf("failed to set unknown values for nested block %q", name))
		}
	}

	return cty.ObjectVal(newVals)
}
