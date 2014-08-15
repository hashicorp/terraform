package schema

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ValueType is an enum of the type that can be represented by a schema.
type ValueType int

const (
	TypeInvalid ValueType = iota
	TypeBoolean
	TypeInt
	TypeString
	TypeList
)

// Schema is used to describe the structure of a value.
type Schema struct {
	// Type is the type of the value and must be one of the ValueType values.
	Type ValueType

	// If one of these is set, then this item can come from the configuration.
	// Both cannot be set. If Optional is set, the value is optional. If
	// Required is set, the value is required.
	Optional bool
	Required bool

	// The fields below relate to diffs: if Computed is true, then the
	// result of this value is computed (unless specified by config).
	// If ForceNew is true
	Computed bool
	ForceNew bool

	// Elem must be either a *Schema or a *Resource only if the Type is
	// TypeList, and represents what the element type is. If it is *Schema,
	// the element type is just a simple value. If it is *Resource, the
	// element type is a complex structure, potentially with its own lifecycle.
	Elem interface{}
}

// schemaMap is a wrapper that adds nice functions on top of schemas.
type schemaMap map[string]*Schema

// Data returns a ResourceData for the given schema, state, and diff.
//
// The diff is optional.
func (m schemaMap) Data(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*ResourceData, error) {
	return nil, nil
}

// Diff returns the diff for a resource given the schema map,
// state, and configuration.
func (m schemaMap) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	result := new(terraform.ResourceDiff)
	result.Attributes = make(map[string]*terraform.ResourceAttrDiff)

	for k, schema := range m {
		var attrD *terraform.ResourceAttrDiff
		var err error

		switch schema.Type {
		case TypeString:
			attrD, err = m.diffString(k, schema, s, c)
		}

		if err != nil {
			return nil, err
		}

		if attrD == nil {
			// There is no diff for this attribute so just carry on.
			continue
		}

		if schema.ForceNew {
			// We require a new one if we have a diff, which we do, so
			// set the flag to true.
			attrD.RequiresNew = true
		}

		result.Attributes[k] = attrD
	}

	return result, nil
}

func (m schemaMap) diffString(
	k string,
	schema *Schema,
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceAttrDiff, error) {
	var old, n string
	if s != nil {
		old = s.Attributes[k]
	}

	computed := false
	v, ok := c.Get(k)
	if !ok {
		// We don't have a value, if it is required then it is an error
		if schema.Required {
			return nil, fmt.Errorf("%s: required field not set", k)
		}

		// We don't have a configuration value.
		if schema.Computed {
			computed = true
		} else {
			return nil, nil
		}
	} else {
		if err := mapstructure.WeakDecode(v, &n); err != nil {
			return nil, fmt.Errorf("%s: %s", k, err)
		}

		if old == n {
			// They're the same value
			return nil, nil
		}
	}

	return &terraform.ResourceAttrDiff{
		Old:         old,
		New:         n,
		NewComputed: computed,
	}, nil
}
