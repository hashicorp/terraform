package schema

import (
	"fmt"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// The functions and methods in this file are concerned with the conversion
// of this package's schema model into the slightly-lower-level schema model
// used by Terraform core for configuration parsing.

// CoreConfigSchema lowers the receiver to the schema model expected by
// Terraform core.
//
// This lower-level model has fewer features than the schema in this package,
// describing only the basic structure of configuration and state values we
// expect. The full schemaMap from this package is still required for full
// validation, handling of default values, etc.
//
// This method presumes a schema that passes InternalValidate, and so may
// panic or produce an invalid result if given an invalid schemaMap.
func (m schemaMap) CoreConfigSchema() *configschema.Block {
	if len(m) == 0 {
		// We return an actual (empty) object here, rather than a nil,
		// because a nil result would mean that we don't have a schema at
		// all, rather than that we have an empty one.
		return &configschema.Block{}
	}

	ret := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{},
		BlockTypes: map[string]*configschema.NestedBlock{},
	}

	for name, schema := range m {
		if schema.Elem == nil {
			ret.Attributes[name] = schema.coreConfigSchemaAttribute()
			continue
		}
		switch schema.Elem.(type) {
		case *Schema, ValueType:
			ret.Attributes[name] = schema.coreConfigSchemaAttribute()
		case *Resource:
			ret.BlockTypes[name] = schema.coreConfigSchemaBlock()
		default:
			// Should never happen for a valid schema
			panic(fmt.Errorf("invalid Schema.Elem %#v; need *Schema or *Resource", schema.Elem))
		}
	}

	return ret
}

// coreConfigSchemaAttribute prepares a configschema.Attribute representation
// of a schema. This is appropriate only for primitives or collections whose
// Elem is an instance of Schema. Use coreConfigSchemaBlock for collections
// whose elem is a whole resource.
func (s *Schema) coreConfigSchemaAttribute() *configschema.Attribute {
	return &configschema.Attribute{
		Type:        s.coreConfigSchemaType(),
		Optional:    s.Optional,
		Required:    s.Required,
		Computed:    s.Computed,
		Sensitive:   s.Sensitive,
		Description: s.Description,
	}
}

// coreConfigSchemaBlock prepares a configschema.NestedBlock representation of
// a schema. This is appropriate only for collections whose Elem is an instance
// of Resource, and will panic otherwise.
func (s *Schema) coreConfigSchemaBlock() *configschema.NestedBlock {
	ret := &configschema.NestedBlock{}
	if nested := s.Elem.(*Resource).CoreConfigSchema(); nested != nil {
		ret.Block = *nested
	}
	switch s.Type {
	case TypeList:
		ret.Nesting = configschema.NestingList
	case TypeSet:
		ret.Nesting = configschema.NestingSet
	case TypeMap:
		ret.Nesting = configschema.NestingMap
	default:
		// Should never happen for a valid schema
		panic(fmt.Errorf("invalid s.Type %s for s.Elem being resource", s.Type))
	}

	ret.MinItems = s.MinItems
	ret.MaxItems = s.MaxItems

	if s.Required && s.MinItems == 0 {
		// configschema doesn't have a "required" representation for nested
		// blocks, but we can fake it by requiring at least one item.
		ret.MinItems = 1
	}

	return ret
}

// coreConfigSchemaType determines the core config schema type that corresponds
// to a particular schema's type.
func (s *Schema) coreConfigSchemaType() cty.Type {
	switch s.Type {
	case TypeString:
		return cty.String
	case TypeBool:
		return cty.Bool
	case TypeInt, TypeFloat:
		// configschema doesn't distinguish int and float, so helper/schema
		// will deal with this as an additional validation step after
		// configuration has been parsed and decoded.
		return cty.Number
	case TypeList, TypeSet, TypeMap:
		var elemType cty.Type
		switch set := s.Elem.(type) {
		case *Schema:
			elemType = set.coreConfigSchemaType()
		case ValueType:
			// This represents a mistake in the provider code, but it's a
			// common one so we'll just shim it.
			elemType = (&Schema{Type: set}).coreConfigSchemaType()
		case *Resource:
			// In practice we don't actually use this for normal schema
			// construction because we construct a NestedBlock in that
			// case instead. See schemaMap.CoreConfigSchema.
			elemType = set.CoreConfigSchema().ImpliedType()
		default:
			if set != nil {
				// Should never happen for a valid schema
				panic(fmt.Errorf("invalid Schema.Elem %#v; need *Schema or *Resource", s.Elem))
			}
			// Some pre-existing schemas assume string as default, so we need
			// to be compatible with them.
			elemType = cty.String
		}
		switch s.Type {
		case TypeList:
			return cty.List(elemType)
		case TypeSet:
			return cty.Set(elemType)
		case TypeMap:
			return cty.Map(elemType)
		default:
			// can never get here in practice, due to the case we're inside
			panic("invalid collection type")
		}
	default:
		// should never happen for a valid schema
		panic(fmt.Errorf("invalid Schema.Type %s", s.Type))
	}
}

// CoreConfigSchema is a convenient shortcut for calling CoreConfigSchema on
// the resource's schema. CoreConfigSchema adds the implicitly required "id"
// attribute for top level resources if it doesn't exist.
func (r *Resource) CoreConfigSchema() *configschema.Block {
	block := r.coreConfigSchema()

	if block.Attributes == nil {
		block.Attributes = map[string]*configschema.Attribute{}
	}

	// Add the implicitly required "id" field if it doesn't exist
	if block.Attributes["id"] == nil {
		block.Attributes["id"] = &configschema.Attribute{
			Type:     cty.String,
			Optional: true,
			Computed: true,
		}
	}

	return block
}

func (r *Resource) coreConfigSchema() *configschema.Block {
	return schemaMap(r.Schema).CoreConfigSchema()
}

// CoreConfigSchema is a convenient shortcut for calling CoreConfigSchema
// on the backends's schema.
func (r *Backend) CoreConfigSchema() *configschema.Block {
	return schemaMap(r.Schema).CoreConfigSchema()
}
