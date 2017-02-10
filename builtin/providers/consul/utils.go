package consul

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// apiAttr is the type used for constants representing well known keys within
// the API that are transmitted back to a resource over an API.
type apiAttr string

// schemaAttr is the type used for constants representing well known keys
// within the schema for a resource.
type schemaAttr string

// sourceFlags represent the ways in which an attribute can be written.  Some
// sources are mutually exclusive, yet other flag combinations are composable.
type sourceFlags int

// typeKey is the lookup mechanism for the generated schema.
type typeKey int

const (
	// sourceUserRequired indicates the parameter must be provided by the user in
	// their configuration.
	sourceUserRequired sourceFlags = 1 << iota

	// sourceUserOptional indicates the parameter may optionally be specified by
	// the user in their configuration.
	sourceUserOptional

	// sourceAPIResult indicates the parameter may only be set by the return of
	// an API call.
	sourceAPIResult

	// sourceLocalFilter indicates the parameter is only used as input to the
	// resource or data source and not to be entered into the state file.
	sourceLocalFilter
)

const (
	// modifyState is a mask that selects all attribute sources that can modify
	// the state (i.e. everything but filters used in data sources).
	modifyState = sourceUserRequired | sourceUserOptional | sourceAPIResult

	// computedAttrMask is a mask that selects source*'s that are Computed in the
	// schema.
	computedAttrMask = sourceAPIResult

	// optionalAttrMask is a mask that selects source*'s that are Optional in the
	// schema.
	optionalAttrMask = sourceAPIResult | sourceLocalFilter

	// requiredAttrMask is a mask that selects source*'s that are Required in the
	// schema.
	requiredAttrMask = sourceUserRequired
)

type typeEntry struct {
	APIName       apiAttr
	APIAliases    []apiAttr
	Source        sourceFlags
	Default       interface{}
	Description   string
	SchemaName    schemaAttr
	Type          schema.ValueType
	ValidateFuncs []interface{}
	SetMembers    map[typeKey]*typeEntry
	ListSchema    map[typeKey]*typeEntry

	// APITest, if returns true, will call APIToState.  The if the value was
	// found, the second return parameter will include the value that should be
	// set in the state store.
	APITest func(*typeEntry, interface{}) (interface{}, bool)

	// APIToState takes the value from APITest and writes it to the attrWriter
	APIToState func(*typeEntry, interface{}, attrWriter) error

	// ConfigRead, if it returns true, returned a value that will be passed to its
	// ConfigUse handler.
	ConfigRead func(*typeEntry, attrReader) (interface{}, bool)

	// ConfigUse takes the value returned from ConfigRead as the second argument
	// and a 3rd optional opaque context argument.
	ConfigUse func(e *typeEntry, v interface{}, target interface{}) error
}

type typeHandlers struct {
	APITest    func(*typeEntry, interface{}) (interface{}, bool)
	APIToState func(*typeEntry, interface{}, attrWriter) error
}

var typeHandlerLookupMap = map[schema.ValueType]*typeHandlers{
	schema.TypeBool: &typeHandlers{
		APITest:    apiTestBool,
		APIToState: apiToStateBool,
	},
	schema.TypeFloat: &typeHandlers{
		APITest:    apiTestFloat64,
		APIToState: apiToStateFloat64,
	},
	schema.TypeList: &typeHandlers{
		APITest:    apiTestList,
		APIToState: apiToStateList,
	},
	schema.TypeMap: &typeHandlers{
		APITest:    apiTestMap,
		APIToState: apiToStateMap,
	},
	schema.TypeSet: &typeHandlers{
		APITest:    apiTestSet,
		APIToState: apiToStateSet,
	},
	schema.TypeString: &typeHandlers{
		APITest:    apiTestString,
		APIToState: apiToStateString,
	},
}

func apiTestBool(e *typeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	v, found := m[string(e.APIName)]
	if found {
		if b, ok := v.(bool); ok {
			return b, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails bool type assertion", e.SchemaName))
		}
	}

	return false, false
}

func apiTestFloat64(e *typeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	v, found := m[string(e.APIName)]
	if found {
		if f, ok := v.(float64); ok {
			return f, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails float64 type assertion", e.SchemaName))
		}
	}
	return 0.0, false
}

func apiTestID(e *typeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	v, _ := apiTestString(e, m)

	// Unconditionally return true so that the call to the APIToState handler can
	// return an error.
	return v, true
}

func apiTestList(e *typeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	names := append([]apiAttr{e.APIName}, e.APIAliases...)
	const defaultListLen = 8
	l := make([]interface{}, 0, defaultListLen)

	var foundName bool
	for _, name := range names {
		v, found := m[string(name)]
		if found {
			foundName = true
			// TODO(sean@): should make a list writer that normalizes v.(type) to a
			// string.  For now we only accept strings and lists.
			switch u := v.(type) {
			case []interface{}:
				l = append(l, u...)
			case string:
				l = append(l, u)
			default:
				panic(fmt.Sprintf("PROVIDER BUG: %q fails list type assertion", e.SchemaName))
			}
		}
	}

	if foundName {
		return l, true
	}

	return []interface{}{}, false
}

func apiTestMap(e *typeEntry, selfRaw interface{}) (interface{}, bool) {
	self := selfRaw.(map[string]interface{})

	v, found := self[string(e.APIName)]
	if found {
		if m, ok := v.(map[string]interface{}); ok {
			return m, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails map type assertion", e.SchemaName))
		}
	}
	return "", false
}

func apiTestSet(e *typeEntry, selfRaw interface{}) (interface{}, bool) {
	self := selfRaw.(map[string]interface{})

	v, found := self[string(e.APIName)]
	if found {
		if m, ok := v.(map[string]interface{}); ok {
			return m, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails map type assertion", e.SchemaName))
		}
	}
	return "", false
}

func apiTestString(e *typeEntry, selfRaw interface{}) (interface{}, bool) {
	self := selfRaw.(map[string]interface{})

	v, found := self[string(e.APIName)]
	if found {
		if s, ok := v.(string); ok {
			return s, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails string type assertion", e.SchemaName))
		}
	}
	return "", false
}

func apiToStateBool(e *typeEntry, v interface{}, w attrWriter) error {
	return w.SetBool(e.SchemaName, v.(bool))
}

func apiToStateID(e *typeEntry, v interface{}, w attrWriter) error {
	s, ok := v.(string)
	if !ok || len(s) == 0 {
		return fmt.Errorf("Unable to set %q's ID to an empty or non-string value: %#v", e.SchemaName, v)
	}

	stateWriter, ok := w.(*attrWriterState)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to SetID with a non-attrWriterState")
	}

	stateWriter.SetID(s)

	return nil
}

func apiToStateFloat64(e *typeEntry, v interface{}, w attrWriter) error {
	f, ok := v.(float64)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a float64", e.SchemaName)
	}

	return w.SetFloat64(e.SchemaName, f)
}

func apiToStateList(e *typeEntry, v interface{}, w attrWriter) error {
	l, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a list", e.SchemaName)
	}

	return w.SetList(e.SchemaName, l)
}

func apiToStateMap(e *typeEntry, v interface{}, w attrWriter) error {
	rawMap, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a map", e.SchemaName)
	}

	mWriter := newMapWriter(make(map[string]interface{}, len(rawMap)))

	// Make a lookup map by API Schema Name
	var setMembersLen int
	if e.SetMembers != nil {
		setMembersLen = len(e.SetMembers)
	}
	apiLookup := make(map[string]*typeEntry, setMembersLen)
	for _, typeEntry := range e.SetMembers {
		apiLookup[string(e.SchemaName)] = typeEntry
	}

	for k, v := range rawMap {
		var usedSchemaHandler bool
		if attrEntry, found := apiLookup[k]; found {
			usedSchemaHandler = true
			if err := attrEntry.APIToState(e, v, mWriter); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Error calling API to state handler on %s: {{err}}", k), err)
			}
		}

		if !usedSchemaHandler {
			if err := mWriter.Set(schemaAttr(k), v); err != nil {
				return errwrap.Wrapf("Unable to store map in state: {{err}}", err)
			}
		}
	}

	return w.SetMap(e.SchemaName, mWriter.ToMap())
}

func apiToStateSet(e *typeEntry, v interface{}, w attrWriter) error {
	s, ok := v.([]map[string]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a set", e.SchemaName)
	}

	set := schema.NewSet(schema.HashResource(nil), nil)
	set.Add(s)

	return w.SetSet(e.SchemaName, set)
}

func apiToStateString(e *typeEntry, v interface{}, w attrWriter) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a float64", e.SchemaName)
	}

	return w.SetString(e.SchemaName, s)
}

func hashMap(in interface{}) int {
	return 0
	m, ok := in.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("PROVIDER BUG: Unable to cast %#v to a map", in))
	}

	keys := make([]string, 0, len(m))
	for k, _ := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	b := &bytes.Buffer{}
	const defaultHashBufSize = 4096
	b.Grow(defaultHashBufSize)

	for _, k := range keys {
		v, found := m[k]
		if !found {
			panic("PROVIDER BUG: race condition: key should not be missing")
		}

		fmt.Fprintf(b, k)
		switch u := v.(type) {
		case string:
			fmt.Fprint(b, u)
		case bool:
			fmt.Fprintf(b, "%t", u)
		case float64:
			fmt.Fprint(b, strconv.FormatFloat(u, 'g', -1, 64))
		case int, uint:
			fmt.Fprintf(b, "%d", u)
		case nil:
		default:
			panic(fmt.Sprintf("Unsupported type %T in map hasher", v))
		}
	}

	return hashcode.String(b.String())
}

func indirect(v interface{}) interface{} {
	switch v.(type) {
	case string:
		return v
	case *string:
		p := v.(*string)
		if p == nil {
			return nil
		}
		return *p
	default:
		return v
	}
}

func (e *typeEntry) LookupDefaultTypeHandler() *typeHandlers {
	h, found := typeHandlerLookupMap[e.Type]
	if !found {
		panic(fmt.Sprintf("PROVIDER BUG: unable to lookup %q's type (%#v)", e.SchemaName, e.Type))
	}

	return h
}

func (e *typeEntry) MustLookupTypeHandler() *typeHandlers {
	h := &typeHandlers{
		APITest:    e.APITest,
		APIToState: e.APIToState,
	}

	defaultHandler := e.LookupDefaultTypeHandler()

	if h.APITest == nil {
		h.APITest = defaultHandler.APITest

		if h.APITest == nil {
			panic(fmt.Sprint("PROVIDER BUG: %v missing APITest method", e.SchemaName))
		}
	}

	if h.APIToState == nil {
		h.APIToState = defaultHandler.APIToState

		if h.APIToState == nil {
			panic(fmt.Sprint("PROVIDER BUG: %v missing APIToState method", e.SchemaName))
		}
	}

	return h
}

// negateBoolToState is a factory function that creates a new function that
// negates whatever the bool is that's passed in as an argument.
func negateBoolToState(fn func(*typeEntry, interface{}, attrWriter) error) func(*typeEntry, interface{}, attrWriter) error {
	return func(e *typeEntry, v interface{}, w attrWriter) error {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("Unable to type assert non-bool value: %#v", v)
		}

		return fn(e, !b, w)
	}
}

// stateSet sets an attribute based on an attrName.  Return an error if the
// Set() to schema.ResourceData fails.
func stateSet(d *schema.ResourceData, attrName schemaAttr, v interface{}) error {
	if err := d.Set(string(attrName), indirect(v)); err != nil {
		return fmt.Errorf("PROVIDER BUG: failed set schema attribute %s to value %#v: %v", attrName, v, err)
	}

	return nil
}

func typeEntryListToSchema(e *typeEntry) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		string(e.SchemaName): e.ToSchema(),
	}
}

func typeEntryMapToResource(in map[typeKey]*typeEntry) *schema.Resource {
	return &schema.Resource{
		Schema: typeEntryMapToSchema(in),
	}
}

func typeEntryMapToSchema(in map[typeKey]*typeEntry) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(in))
	for _, e := range in {
		out[string(e.SchemaName)] = e.ToSchema()
	}

	return out
}

func (e *typeEntry) Validate() {
	if e.Source&sourceAPIResult != 0 && e.Type == schema.TypeSet {
		panic(fmt.Sprintf("PROVIDER BUG: %s can not be computed and of type Set", e.SchemaName))
	}

	if e.Source&sourceLocalFilter != 0 {
		if e.ConfigRead == nil {
			panic(fmt.Sprintf("PROVIDER BUG: %s can not be configured as a local filter and be missing a config read handler", e.SchemaName))
		}

		if e.ConfigUse == nil {
			panic(fmt.Sprintf("PROVIDER BUG: %s can not be configured as a local filter and be missing a config use handler", e.SchemaName))
		}
	}

	if len(e.SetMembers) != 0 && !(e.Type == schema.TypeSet || e.Type == schema.TypeMap) {
		panic(fmt.Sprintf("PROVIDER BUG: %s is not of type Set but has SetMembers set", e.SchemaName))
	}

	if e.Source&(sourceUserRequired|sourceAPIResult) == (sourceUserRequired | sourceAPIResult) {
		panic(fmt.Sprintf("PROVIDER BUG: %#v and %#v are mutually exclusive Source flags", sourceUserRequired, sourceAPIResult))
	}
}

func (e *typeEntry) ToSchema() *schema.Schema {
	e.Validate()

	attr := &schema.Schema{
		Computed:    e.Source&computedAttrMask != 0,
		Default:     e.Default,
		Description: e.Description,
		Optional:    e.Source&optionalAttrMask != 0,
		Required:    e.Source&requiredAttrMask != 0,
		Type:        e.Type,
		// ValidateFunc: e.MakeValidationFunc(),
	}

	// Fixup the type: use the real type vs a surrogate type
	switch e.Type {
	case schema.TypeList:
		if e.ListSchema == nil {
			attr.Elem = &schema.Schema{
				Type: schema.TypeString,
			}
		} else {
			attr.Elem = typeEntryMapToResource(e.ListSchema)
		}

	case schema.TypeSet:
		attr.Elem = &schema.Resource{
			Schema: typeEntryMapToSchema(e.SetMembers),
		}
	}

	return attr
}

func mapStringToMapInterface(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
