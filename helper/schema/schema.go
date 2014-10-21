// schema is a high-level framework for easily writing new providers
// for Terraform. Usage of schema is recommended over attempting to write
// to the low-level plugin interfaces manually.
//
// schema breaks down provider creation into simple CRUD operations for
// resources. The logic of diffing, destroying before creating, updating
// or creating, etc. is all handled by the framework. The plugin author
// only needs to implement a configuration schema and the CRUD operations and
// everything else is meant to just work.
//
// A good starting point is to view the Provider structure.
package schema

import (
	"fmt"
	"reflect"
	"sort"
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
//
// Read the documentation of the struct elements for important details.
type Schema struct {
	// Type is the type of the value and must be one of the ValueType values.
	//
	// This type not only determines what type is expected/valid in configuring
	// this value, but also what type is returned when ResourceData.Get is
	// called. The types returned by Get are:
	//
	//   TypeBool - bool
	//   TypeInt - int
	//   TypeString - string
	//   TypeList - []interface{}
	//   TypeMap - map[string]interface{}
	//   TypeSet - *schema.Set
	//
	Type ValueType

	// If one of these is set, then this item can come from the configuration.
	// Both cannot be set. If Optional is set, the value is optional. If
	// Required is set, the value is required.
	//
	// One of these must be set if the value is not computed. That is:
	// value either comes from the config, is computed, or is both.
	Optional bool
	Required bool

	// If this is non-nil, then this will be a default value that is used
	// when this item is not set in the configuration/state.
	//
	// DefaultFunc can be specified if you want a dynamic default value.
	// Only one of Default or DefaultFunc can be set.
	//
	// If Required is true above, then Default cannot be set. DefaultFunc
	// can be set with Required. If the DefaultFunc returns nil, then there
	// will no default and the user will be asked to fill it in.
	//
	// If either of these is set, then the user won't be asked for input
	// for this key if the default is not nil.
	Default     interface{}
	DefaultFunc SchemaDefaultFunc

	// Description is used as the description for docs or asking for user
	// input. It should be relatively short (a few sentences max) and should
	// be formatted to fit a CLI.
	Description string

	// InputDefault is the default value to use for when inputs are requested.
	// This differs from Default in that if Default is set, no input is
	// asked for. If Input is asked, this will be the default value offered.
	InputDefault string

	// The fields below relate to diffs.
	//
	// If Computed is true, then the result of this value is computed
	// (unless specified by config) on creation.
	//
	// If ForceNew is true, then a change in this resource necessitates
	// the creation of a new resource.
	//
	// StateFunc is a function called to change the value of this before
	// storing it in the state (and likewise before comparing for diffs).
	// The use for this is for example with large strings, you may want
	// to simply store the hash of it.
	Computed  bool
	ForceNew  bool
	StateFunc SchemaStateFunc

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
	//
	// NOTE: This currently does not work.
	ComputedWhen []string
}

// SchemaDefaultFunc is a function called to return a default value for
// a field.
type SchemaDefaultFunc func() (interface{}, error)

// SchemaSetFunc is a function that must return a unique ID for the given
// element. This unique ID is used to store the element in a hash.
type SchemaSetFunc func(interface{}) int

// SchemaStateFunc is a function used to convert some type to a string
// to be stored in the state.
type SchemaStateFunc func(interface{}) string

func (s *Schema) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

func (s *Schema) finalizeDiff(
	d *terraform.ResourceAttrDiff) *terraform.ResourceAttrDiff {
	if d == nil {
		return d
	}

	if d.NewRemoved {
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
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*ResourceData, error) {
	return &ResourceData{
		schema: m,
		state:  s,
		diff:   d,
	}, nil
}

// Diff returns the diff for a resource given the schema map,
// state, and configuration.
func (m schemaMap) Diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	result := new(terraform.InstanceDiff)
	result.Attributes = make(map[string]*terraform.ResourceAttrDiff)

	d := &ResourceData{
		schema:  m,
		state:   s,
		config:  c,
		diffing: true,
	}

	for k, schema := range m {
		err := m.diff(k, schema, result, d, false)
		if err != nil {
			return nil, err
		}
	}

	// If the diff requires a new resource, then we recompute the diff
	// so we have the complete new resource diff, and preserve the
	// RequiresNew fields where necessary so the user knows exactly what
	// caused that.
	if result.RequiresNew() {
		// Create the new diff
		result2 := new(terraform.InstanceDiff)
		result2.Attributes = make(map[string]*terraform.ResourceAttrDiff)

		// Reset the data to not contain state
		d.state = nil

		// Perform the diff again
		for k, schema := range m {
			err := m.diff(k, schema, result2, d, false)
			if err != nil {
				return nil, err
			}
		}

		// Force all the fields to not force a new since we know what we
		// want to force new.
		for k, attr := range result2.Attributes {
			if attr == nil {
				continue
			}

			if attr.RequiresNew {
				attr.RequiresNew = false
			}

			if s != nil {
				attr.Old = s.Attributes[k]
			}
		}

		// Now copy in all the requires new diffs...
		for k, attr := range result.Attributes {
			if attr == nil {
				continue
			}

			newAttr, ok := result2.Attributes[k]
			if !ok {
				newAttr = attr
			}

			if attr.RequiresNew {
				newAttr.RequiresNew = true
			}

			result2.Attributes[k] = newAttr
		}

		// And set the diff!
		result = result2
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

// Input implements the terraform.ResourceProvider method by asking
// for input for required configuration keys that don't have a value.
func (m schemaMap) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	keys := make([]string, 0, len(m))
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]

		// Skip things that don't require config, if that is even valid
		// for a provider schema.
		if !v.Required && !v.Optional {
			continue
		}

		// Skip things that have a value of some sort already
		if _, ok := c.Raw[k]; ok {
			continue
		}

		// Skip if it has a default
		if v.Default != nil {
			continue
		}
		if f := v.DefaultFunc; f != nil {
			value, err := f()
			if err != nil {
				return nil, fmt.Errorf(
					"%s: error loading default: %s", k, err)
			}
			if value != nil {
				continue
			}
		}

		var value interface{}
		var err error
		switch v.Type {
		case TypeBool:
			fallthrough
		case TypeInt:
			fallthrough
		case TypeString:
			value, err = m.inputString(input, k, v)
		default:
			panic(fmt.Sprintf("Unknown type for input: %s", v.Type))
		}

		if err != nil {
			return nil, fmt.Errorf(
				"%s: %s", k, err)
		}

		c.Config[k] = value
	}

	return c, nil
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

		if !v.Required && !v.Optional && !v.Computed {
			return fmt.Errorf("%s: One of optional, required, or computed must be set", k)
		}

		if v.Computed && v.Default != nil {
			return fmt.Errorf("%s: Default must be nil if computed", k)
		}

		if v.Required && v.Default != nil {
			return fmt.Errorf("%s: Default cannot be set with Required", k)
		}

		if len(v.ComputedWhen) > 0 && !v.Computed {
			return fmt.Errorf("%s: ComputedWhen can only be set with Computed", k)
		}

		if v.Type == TypeList || v.Type == TypeSet {
			if v.Elem == nil {
				return fmt.Errorf("%s: Elem must be set for lists", k)
			}

			if v.Default != nil {
				return fmt.Errorf("%s: Default is not valid for lists or sets", k)
			}

			if v.Type == TypeList && v.Set != nil {
				return fmt.Errorf("%s: Set can only be set for TypeSet", k)
			} else if v.Type == TypeSet && v.Set == nil {
				return fmt.Errorf("%s: Set must be set", k)
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
	diff *terraform.InstanceDiff,
	d *ResourceData,
	all bool) error {
	var err error
	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		err = m.diffString(k, schema, diff, d, all)
	case TypeList:
		err = m.diffList(k, schema, diff, d, all)
	case TypeMap:
		err = m.diffMap(k, schema, diff, d, all)
	case TypeSet:
		err = m.diffSet(k, schema, diff, d, all)
	default:
		err = fmt.Errorf("%s: unknown type %#v", k, schema.Type)
	}

	return err
}

func (m schemaMap) diffList(
	k string,
	schema *Schema,
	diff *terraform.InstanceDiff,
	d *ResourceData,
	all bool) error {
	o, n, _, computedList := d.diffChange(k)
	nSet := n != nil

	// If we have an old value, but no new value set but we're computed,
	// then nothing has changed.
	if o != nil && n == nil && schema.Computed {
		return nil
	}

	if o == nil {
		o = []interface{}{}
	}
	if n == nil {
		n = []interface{}{}
	}
	if s, ok := o.(*Set); ok {
		o = s.List()
	}
	if s, ok := n.(*Set); ok {
		n = s.List()
	}
	os := o.([]interface{})
	vs := n.([]interface{})

	// If the new value was set, and the two are equal, then we're done.
	// We have to do this check here because sets might be NOT
	// reflect.DeepEqual so we need to wait until we get the []interface{}
	if !all && nSet && reflect.DeepEqual(os, vs) {
		return nil
	}

	// Get the counts
	oldLen := len(os)
	newLen := len(vs)
	oldStr := strconv.FormatInt(int64(oldLen), 10)

	// If the whole list is computed, then say that the # is computed
	if computedList {
		diff.Attributes[k+".#"] = &terraform.ResourceAttrDiff{
			Old:         oldStr,
			NewComputed: true,
		}
		return nil
	}

	// If the counts are not the same, then record that diff
	changed := oldLen != newLen
	computed := oldLen == 0 && newLen == 0 && schema.Computed
	if changed || computed || all {
		countSchema := &Schema{
			Type:     TypeInt,
			Computed: schema.Computed,
			ForceNew: schema.ForceNew,
		}

		newStr := ""
		if !computed {
			newStr = strconv.FormatInt(int64(newLen), 10)
		} else {
			oldStr = ""
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
			err := m.diff(subK, &t2, diff, d, all)
			if err != nil {
				return err
			}
		}
	case *Resource:
		// This is a complex resource
		for i := 0; i < maxLen; i++ {
			for k2, schema := range t.Schema {
				subK := fmt.Sprintf("%s.%d.%s", k, i, k2)
				err := m.diff(subK, schema, diff, d, all)
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
	diff *terraform.InstanceDiff,
	d *ResourceData,
	all bool) error {
	prefix := k + "."

	// First get all the values from the state
	var stateMap, configMap map[string]string
	o, n, _, _ := d.diffChange(k)
	if err := mapstructure.WeakDecode(o, &stateMap); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}
	if err := mapstructure.WeakDecode(n, &configMap); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// If the new map is nil and we're computed, then ignore it.
	if n == nil && schema.Computed {
		return nil
	}

	// Now we compare, preferring values from the config map
	for k, v := range configMap {
		old := stateMap[k]
		delete(stateMap, k)

		if old == v && !all {
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
	diff *terraform.InstanceDiff,
	d *ResourceData,
	all bool) error {
	if !all {
		// This is a bit strange, but we expect the entire set to be in the diff,
		// so we first diff the set normally but with a new diff. Then, if
		// there IS any change, we just set the change to the entire list.
		tempD := new(terraform.InstanceDiff)
		tempD.Attributes = make(map[string]*terraform.ResourceAttrDiff)
		if err := m.diffList(k, schema, tempD, d, false); err != nil {
			return err
		}

		// If we had no changes, then we're done
		if tempD.Empty() {
			return nil
		}
	}

	// We have changes, so re-run the diff, but set a flag to force
	// getting all diffs, even if there is no change.
	return m.diffList(k, schema, diff, d, true)
}

func (m schemaMap) diffString(
	k string,
	schema *Schema,
	diff *terraform.InstanceDiff,
	d *ResourceData,
	all bool) error {
	var originalN interface{}
	var os, ns string
	o, n, _, _ := d.diffChange(k)
	if n == nil {
		n = schema.Default
		if schema.DefaultFunc != nil {
			var err error
			n, err = schema.DefaultFunc()
			if err != nil {
				return fmt.Errorf("%s, error loading default: %s", err)
			}
		}
	}
	if schema.StateFunc != nil {
		originalN = n
		n = schema.StateFunc(n)
	}
	if err := mapstructure.WeakDecode(o, &os); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}
	if err := mapstructure.WeakDecode(n, &ns); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	if os == ns && !all {
		// They're the same value. If there old value is not blank or we
		// have an ID, then return right away since we're already setup.
		if os != "" || d.Id() != "" {
			return nil
		}

		// Otherwise, only continue if we're computed
		if !schema.Computed {
			return nil
		}
	}

	removed := false
	if o != nil && n == nil {
		removed = true
	}
	if removed && schema.Computed {
		return nil
	}

	diff.Attributes[k] = schema.finalizeDiff(&terraform.ResourceAttrDiff{
		Old:        os,
		New:        ns,
		NewExtra:   originalN,
		NewRemoved: removed,
	})

	return nil
}

func (m schemaMap) inputString(
	input terraform.UIInput,
	k string,
	schema *Schema) (interface{}, error) {
	result, err := input.Input(&terraform.InputOpts{
		Id:          k,
		Query:       k,
		Description: schema.Description,
		Default:     schema.InputDefault,
	})

	return result, err
}

func (m schemaMap) validate(
	k string,
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	raw, ok := c.Get(k)
	if !ok && schema.DefaultFunc != nil {
		// We have a dynamic default. Check if we have a value.
		var err error
		raw, err = schema.DefaultFunc()
		if err != nil {
			return nil, []error{fmt.Errorf(
				"%s, error loading default: %s", k, err)}
		}

		// We're okay as long as we had a value set
		ok = raw != nil
	}
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

func (m schemaMap) validateMap(
	k string,
	raw interface{},
	schema *Schema,
	c *terraform.ResourceConfig) ([]string, []error) {
	// We use reflection to verify the slice because you can't
	// case to []interface{} unless the slice is exactly that type.
	rawV := reflect.ValueOf(raw)
	switch rawV.Kind() {
	case reflect.Map:
	case reflect.Slice:
	default:
		return nil, []error{fmt.Errorf(
			"%s: should be a map", k)}
	}

	// If it is not a slice, it is valid
	if rawV.Kind() != reflect.Slice {
		return nil, nil
	}

	// It is a slice, verify that all the elements are maps
	raws := make([]interface{}, rawV.Len())
	for i, _ := range raws {
		raws[i] = rawV.Index(i).Interface()
	}

	for _, raw := range raws {
		v := reflect.ValueOf(raw)
		if v.Kind() != reflect.Map {
			return nil, []error{fmt.Errorf(
				"%s: should be a map", k)}
		}
	}

	return nil, nil
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
	if c.IsComputed(k) {
		// If the key is being computed, then it is not an error
		return nil, nil
	}

	switch schema.Type {
	case TypeSet:
		fallthrough
	case TypeList:
		return m.validateList(k, raw, schema, c)
	case TypeMap:
		return m.validateMap(k, raw, schema, c)
	case TypeBool:
		// Verify that we can parse this as the correct type
		var n bool
		if err := mapstructure.WeakDecode(raw, &n); err != nil {
			return nil, []error{err}
		}
	case TypeInt:
		// Verify that we can parse this as an int
		var n int
		if err := mapstructure.WeakDecode(raw, &n); err != nil {
			return nil, []error{err}
		}
	case TypeString:
		// Verify that we can parse this as a string
		var n string
		if err := mapstructure.WeakDecode(raw, &n); err != nil {
			return nil, []error{err}
		}
	default:
		panic(fmt.Sprintf("Unknown validation type: %s", schema.Type))
	}

	return nil, nil
}
