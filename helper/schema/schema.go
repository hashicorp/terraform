package schema

import (
	"fmt"
	"reflect"
	"strconv"
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
	TypeMap
	TypeSet
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

	// The fields below relate to diffs.
	//
	// If Computed is true, then the result of this value is computed
	// (unless specified by config) on creation.
	//
	// If ForceNew is true, then a change in this resource necessitates
	// the creation of a new resource.
	Computed bool
	ForceNew bool

	// The following fields are only set for a TypeList or TypeSet Type.
	//
	// Elem must be either a *Schema or a *Resource only if the Type is
	// TypeList, and represents what the element type is. If it is *Schema,
	// the element type is just a simple value. If it is *Resource, the
	// element type is a complex structure, potentially with its own lifecycle.
	Elem interface{}

	// The follow fields are only valid for a TypeSet type.
	//
	// Set defines a function to determine the unique ID of an item so that
	// a proper set can be built.
	Set SchemaSetFunc

	// ComputedWhen is a set of queries on the configuration. Whenever any
	// of these things is changed, it will require a recompute (this requires
	// that Computed is set to true).
	ComputedWhen []string
}

// SchemaSetFunc is a function that must return a unique ID for the given
// element. This unique ID is used to store the element in a hash.
type SchemaSetFunc func(a interface{}) int

func (s *Schema) finalizeDiff(
	d *terraform.ResourceAttrDiff) *terraform.ResourceAttrDiff {
	if d == nil {
		return d
	}

	if s.Computed {
		if d.Old != "" && d.New == "" {
			// This is a computed value with an old value set already,
			// just let it go.
			return nil
		}

		if d.New == "" {
			// Computed attribute without a new value set
			d.NewComputed = true
		}
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

	d := &ResourceData{
		schema:  m,
		state:   s,
		config:  c,
		diffing: true,
	}

	for k, schema := range m {
		err := m.diff(k, schema, result, d)
		if err != nil {
			return nil, err
		}
	}

	// Remove any nil diffs just to keep things clean
	for k, v := range result.Attributes {
		if v == nil {
			delete(result.Attributes, k)
		}
	}

	// Go through and detect all of the ComputedWhens now that we've
	// finished the diff.
	// TODO

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

// InternalValidate validates the format of this schema. This should be called
// from a unit test (and not in user-path code) to verify that a schema
// is properly built.
func (m schemaMap) InternalValidate() error {
	for k, v := range m {
		if v.Type == TypeInvalid {
			return fmt.Errorf("%s: Type must be specified", k)
		}

		if v.Optional && v.Required {
			return fmt.Errorf("%s: Optional or Required must be set, not both", k)
		}

		if v.Required && v.Computed {
			return fmt.Errorf("%s: Cannot be both Required and Computed", k)
		}

		if len(v.ComputedWhen) > 0 && !v.Computed {
			return fmt.Errorf("%s: ComputedWhen can only be set with Computed", k)
		}

		if v.Type == TypeList {
			if v.Elem == nil {
				return fmt.Errorf("%s: Elem must be set for lists", k)
			}

			// TODO: test
			if v.Set != nil {
				return fmt.Errorf("%s: Set can only be set for TypeSet", k)
			}

			switch t := v.Elem.(type) {
			case *Resource:
				if err := t.InternalValidate(); err != nil {
					return err
				}
			case *Schema:
				bad := t.Computed || t.Optional || t.Required
				if bad {
					return fmt.Errorf(
						"%s: Elem must have only Type set", k)
				}
			}
		}
	}

	return nil
}

func (m schemaMap) diff(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	d *ResourceData) error {
	var err error
	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		err = m.diffString(k, schema, diff, d)
	case TypeList:
		err = m.diffList(k, schema, diff, d)
	case TypeMap:
		err = m.diffMap(k, schema, diff, d)
	case TypeSet:
		err = m.diffSet(k, schema, diff, d)
	default:
		err = fmt.Errorf("%s: unknown type %s", k, schema.Type)
	}

	return err
}

func (m schemaMap) diffList(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	d *ResourceData) error {
	o, n, _ := d.diffChange(k)
	os := o.([]interface{})
	vs := n.([]interface{})

	// Get the counts
	oldLen := len(os)
	newLen := len(vs)

	// If the counts are not the same, then record that diff
	changed := oldLen != newLen
	computed := oldLen == 0 && newLen == 0 && schema.Computed
	if changed || computed {
		countSchema := &Schema{
			Type:     TypeInt,
			Computed: schema.Computed,
			ForceNew: schema.ForceNew,
		}

		oldStr := ""
		newStr := ""
		if !computed {
			oldStr = strconv.FormatInt(int64(oldLen), 10)
			newStr = strconv.FormatInt(int64(newLen), 10)
		}

		diff.Attributes[k+".#"] = countSchema.finalizeDiff(&terraform.ResourceAttrDiff{
			Old: oldStr,
			New: newStr,
		})
	}

	// Figure out the maximum
	maxLen := oldLen
	if newLen > maxLen {
		maxLen = newLen
	}

	switch t := schema.Elem.(type) {
	case *Schema:
		// Copy the schema so that we can set Computed/ForceNew from
		// the parent schema (the TypeList).
		t2 := *t
		t2.ForceNew = schema.ForceNew

		// This is just a primitive element, so go through each and
		// just diff each.
		for i := 0; i < maxLen; i++ {
			subK := fmt.Sprintf("%s.%d", k, i)
			err := m.diff(subK, &t2, diff, d)
			if err != nil {
				return err
			}
		}
	case *Resource:
		// This is a complex resource
		for i := 0; i < maxLen; i++ {
			for k2, schema := range t.Schema {
				subK := fmt.Sprintf("%s.%d.%s", k, i, k2)
				err := m.diff(subK, schema, diff, d)
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

func (m schemaMap) diffMap(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	d *ResourceData) error {
	//elemSchema := &Schema{Type: TypeString}
	prefix := k + "."

	// First get all the values from the state
	var stateMap, configMap map[string]string
	o, n, _ := d.diffChange(k)
	if err := mapstructure.WeakDecode(o, &stateMap); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}
	if err := mapstructure.WeakDecode(n, &configMap); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// Now we compare, preferring values from the config map
	for k, v := range configMap {
		old := stateMap[k]
		delete(stateMap, k)

		if old == v {
			continue
		}

		diff.Attributes[prefix+k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
			Old: old,
			New: v,
		})
	}
	for k, v := range stateMap {
		diff.Attributes[prefix+k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
			Old:        v,
			NewRemoved: true,
		})
	}

	return nil
}

func (m schemaMap) diffSet(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	d *ResourceData) error {
	return nil
}

func (m schemaMap) diffString(
	k string,
	schema *Schema,
	diff *terraform.ResourceDiff,
	d *ResourceData) error {
	var os, ns string
	o, n, _ := d.diffChange(k)
	if err := mapstructure.WeakDecode(o, &os); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}
	if err := mapstructure.WeakDecode(n, &ns); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	if os == ns {
		// They're the same value, return no diff as long as we're not
		// computing a new value.
		if os != "" || !schema.Computed {
			return nil
		}
	}

	diff.Attributes[k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
		Old: os,
		New: ns,
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
				"%s: required field is not set", k)}
		}

		return nil, nil
	}

	if !schema.Required && !schema.Optional {
		// This is a computed-only field
		return nil, []error{fmt.Errorf(
			"%s: this field cannot be set", k)}
	}

	return m.validatePrimitive(k, raw, schema, c)
}

func (m schemaMap) validateList(
	k string,
	raw interface{},
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	// We use reflection to verify the slice because you can't
	// case to []interface{} unless the slice is exactly that type.
	rawV := reflect.ValueOf(raw)
	if rawV.Kind() != reflect.Slice {
		return nil, []error{fmt.Errorf(
			"%s: should be a list", k)}
	}

	// Now build the []interface{}
	raws := make([]interface{}, rawV.Len())
	for i, _ := range raws {
		raws[i] = rawV.Index(i).Interface()
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
