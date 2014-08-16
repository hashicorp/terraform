package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// ValueType is an enum of the type that can be represented by a schema.
type ValueType int

const (
	TypeInvalid ValueType = iota
	TypeBool
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

	// The following fields are only set for a TypeList Type.
	//
	// Elem must be either a *Schema or a *Resource only if the Type is
	// TypeList, and represents what the element type is. If it is *Schema,
	// the element type is just a simple value. If it is *Resource, the
	// element type is a complex structure, potentially with its own lifecycle.
	Elem interface{}
}

func (s *Schema) finalizeDiff(
	d *terraform.ResourceAttrDiff) *terraform.ResourceAttrDiff {
	if d == nil {
		return d
	}

	if s.Computed && d.New == "" {
		// Computed attribute without a new value set
		d.NewComputed = true
	}

	if s.ForceNew {
		// Force new, set it to true in the diff
		d.RequiresNew = true
	}

	return d
}

// schemaMap is a wrapper that adds nice functions on top of schemas.
type schemaMap map[string]*Schema

// Data returns a ResourceData for the given schema, state, and diff.
//
// The diff is optional.
func (m schemaMap) Data(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*ResourceData, error) {
	return &ResourceData{
		schema: m,
		state:  s,
		diff:   d,
	}, nil
}

// Diff returns the diff for a resource given the schema map,
// state, and configuration.
func (m schemaMap) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	result := new(terraform.ResourceDiff)
	result.Attributes = make(map[string]*terraform.ResourceAttrDiff)

	for k, schema := range m {
		err := m.diff(k, schema, result, s, c)
		if err != nil {
			return nil, err
		}
	}

	if result.Empty() {
		// If we don't have any diff elements, just return nil
		return nil, nil
	}

	return result, nil
}

// Validate validates the configuration against this schema mapping.
func (m schemaMap) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return m.validateObject("", m, c)
}

func (m schemaMap) diff(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) error {
	var err error
	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		err = m.diffString(k, schema, diff, s, c)
	case TypeList:
		err = m.diffList(k, schema, diff, s, c)
	default:
		err = fmt.Errorf("%s: unknown type %s", k, schema.Type)
	}

	return err
}

func (m schemaMap) diffList(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) error {
	v, ok := c.Get(k)
	if !ok {
		// We don't have a value, if it is required then it is an error
		if schema.Required {
			return fmt.Errorf("%s: required field not set", k)
		}

		// We don't have a configuration value.
		if !schema.Computed {
			return nil
		}
	}

	vs, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("%s: must be a list", k)
	}

	// If this field is required, then it must also be non-empty
	if len(vs) == 0 && schema.Required {
		return fmt.Errorf("%s: required field is not set", k)
	}

	// Diff the count no matter what
	countSchema := &Schema{
		Type:     TypeInt,
		ForceNew: schema.ForceNew,
	}
	m.diffString(k+".#", countSchema, diff, s, c)

	switch t := schema.Elem.(type) {
	case *Schema:
		// Copy the schema so that we can set Computed/ForceNew from
		// the parent schema (the TypeList).
		t2 := *t
		t2.Computed = schema.Computed
		t2.ForceNew = schema.ForceNew

		// This is just a primitive element, so go through each and
		// just diff each.
		for i, _ := range vs {
			subK := fmt.Sprintf("%s.%d", k, i)
			err := m.diff(subK, &t2, diff, s, c)
			if err != nil {
				return err
			}
		}
	case *Resource:
		// This is a complex resource
		for i, _ := range vs {
			for k2, schema := range t.Schema {
				subK := fmt.Sprintf("%s.%d.%s", k, i, k2)
				err := m.diff(subK, schema, diff, s, c)
				if err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("%s: unknown element type (internal)", k)
	}

	return nil
}

func (m schemaMap) diffString(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) error {
	var old, n string
	if s != nil {
		old = s.Attributes[k]
	}

	v, ok := c.Get(k)
	if !ok {
		// We don't have a value, if it is required then it is an error
		if schema.Required {
			return fmt.Errorf("%s: required field not set", k)
		}

		// We don't have a configuration value.
		if !schema.Computed {
			return nil
		}
	} else {
		if err := mapstructure.WeakDecode(v, &n); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}

		if old == n {
			// They're the same value
			return nil
		}
	}

	diff.Attributes[k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
		Old: old,
		New: n,
	})

	return nil
}

func (m schemaMap) diffPrimitive(
	k string,
	nraw interface{},
	schema *Schema,
	diff *terraform.ResourceDiff,
	s *terraform.ResourceState) error {
	var old, n string
	if s != nil {
		old = s.Attributes[k]
	}

	if err := mapstructure.WeakDecode(nraw, &n); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	if old == n {
		// They're the same value
		return nil
	}

	diff.Attributes[k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
		Old: old,
		New: n,
	})

	return nil
}

func (m schemaMap) validate(
	k string,
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	raw, ok := c.Get(k)
	if !ok {
		if schema.Required {
			return nil, []error{fmt.Errorf(
				"%s: required field is not set")}
		}

		return nil, nil
	}

	return m.validatePrimitive(k, raw, schema, c)
}

func (m schemaMap) validateList(
	k string,
	raw interface{},
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	raws, ok := raw.([]interface{})
	if !ok {
		return nil, []error{fmt.Errorf(
			"%s: should be list", k)}
	}

	var ws []string
	var es []error
	for i, raw := range raws {
		key := fmt.Sprintf("%s.%d", k, i)

		var ws2 []string
		var es2 []error
		switch t := schema.Elem.(type) {
		case *Resource:
			// This is a sub-resource
			ws2, es2 = m.validateObject(key, t.Schema, c)
		case *Schema:
			// This is some sort of primitive
			ws2, es2 = m.validatePrimitive(key, raw, t, c)
		}

		if len(ws2) > 0 {
			ws = append(ws, ws2...)
		}
		if len(es2) > 0 {
			es = append(es, es2...)
		}
	}

	return ws, es
}

func (m schemaMap) validateObject(
	k string,
	schema map[string]*Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	var ws []string
	var es []error
	for subK, s := range schema {
		key := subK
		if k != "" {
			key = fmt.Sprintf("%s.%s", k, subK)
		}

		ws2, es2 := m.validate(key, s, c)
		if len(ws2) > 0 {
			ws = append(ws, ws2...)
		}
		if len(es2) > 0 {
			es = append(es, es2...)
		}
	}

	// Detect any extra/unknown keys and report those as errors.
	prefix := k + "."
	for configK, _ := range c.Raw {
		if k != "" {
			if !strings.HasPrefix(configK, prefix) {
				continue
			}

			configK = configK[len(prefix):]
		}

		if _, ok := schema[configK]; !ok {
			es = append(es, fmt.Errorf(
				"%s: invalid or unknown key: %s", k, configK))
		}
	}

	return ws, es
}

func (m schemaMap) validatePrimitive(
	k string,
	raw interface{},
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	switch schema.Type {
	case TypeList:
		return m.validateList(k, raw, schema, c)
	case TypeInt:
		// Verify that we can parse this as an int
		var n int
		if err := mapstructure.WeakDecode(raw, &n); err != nil {
			return nil, []error{err}
		}
	}

	return nil, nil
}
