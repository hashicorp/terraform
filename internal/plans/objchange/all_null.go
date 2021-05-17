package objchange

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// AllBlockAttributesNull constructs a non-null cty.Value of the object type implied
// by the given schema that has all of its leaf attributes set to null and all
// of its nested block collections set to zero-length.
//
// This simulates what would result from decoding an empty configuration block
// with the given schema, except that it does not produce errors
func AllBlockAttributesNull(schema *configschema.Block) cty.Value {
	// "All attributes null" happens to be the definition of EmptyValue for
	// a Block, so we can just delegate to that.
	return schema.EmptyValue()
}

// AllAttributesNull returns a cty.Value of the object type implied by the given
// attriubutes that has all of its leaf attributes set to null.
func AllAttributesNull(attrs map[string]*configschema.Attribute) cty.Value {
	newAttrs := make(map[string]cty.Value, len(attrs))

	for name, attr := range attrs {
		if attr.NestedType != nil {
			newAttrs[name] = AllAttributesNull(attr.NestedType.Attributes)
		} else {
			newAttrs[name] = cty.NullVal(attr.Type)
		}
	}
	return cty.ObjectVal(newAttrs)
}
