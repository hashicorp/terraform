package objchange

import (
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
	// "All attributes null" happens to be the definition of EmptyValue for
	// a Block, so we can just delegate to that.
	return schema.EmptyValue()
}
