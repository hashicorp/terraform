package objchange

import (
	"fmt"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// AllAttributesNull constructs a non-null cty.Value of the object type implied
// by the given schema that has all of its leaf attributes set to null and all
// of its nested block collections set to zero-length.
//
// This simulates what would result from decoding an empty configuration block
// with the given schema, except that it does not produce errors
func AllAttributesNull(schema *configschema.Block) cty.Value {
	vals := make(map[string]cty.Value)
	ty := schema.ImpliedType()

	for name := range schema.Attributes {
		aty := ty.AttributeType(name)
		vals[name] = cty.NullVal(aty)
	}

	for name, blockS := range schema.BlockTypes {
		aty := ty.AttributeType(name)

		switch blockS.Nesting {
		case configschema.NestingSingle:
			// NestingSingle behaves like an object attribute, which decodes
			// as null when it's not present in configuration.
			vals[name] = cty.NullVal(aty)
		default:
			// All other nesting types decode as "empty" when not present, but
			// empty values take different forms depending on the type.
			switch {
			case aty.IsListType():
				vals[name] = cty.ListValEmpty(aty.ElementType())
			case aty.IsSetType():
				vals[name] = cty.SetValEmpty(aty.ElementType())
			case aty.IsMapType():
				vals[name] = cty.MapValEmpty(aty.ElementType())
			case aty.Equals(cty.DynamicPseudoType):
				// We use DynamicPseudoType in situations where there's a
				// nested attribute of DynamicPseudoType, since the schema
				// system cannot predict the final type until it knows exactly
				// how many elements there will be. However, since we're
				// trying to behave as if there are _no_ elements, we know
				// we're producing either an empty tuple or empty object
				// and just need to distinguish these two cases.
				switch blockS.Nesting {
				case configschema.NestingList:
					vals[name] = cty.EmptyTupleVal
				case configschema.NestingMap:
					vals[name] = cty.EmptyObjectVal
				}
			}
		}

		// By the time we get down here we should always have set a value.
		// If not, that suggests a missing case in the above switches.
		if _, ok := vals[name]; !ok {
			panic(fmt.Sprintf("failed to create empty value for nested block %q", name))
		}
	}

	return cty.ObjectVal(vals)
}
