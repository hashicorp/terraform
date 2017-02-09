package consul

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// _APIAttr is the type used for constants representing well known keys within
// the API that are transmitted back to a resource over an API.
type _APIAttr string

// _SchemaAttr is the type used for constants representing well known keys
// within the schema for a resource.
type _SchemaAttr string

// _SourceFlags represent the ways in which an attribute can be written.  Some
// sources are mutually exclusive, yet other flag combinations are composable.
type _SourceFlags int

// _TypeKey is the lookup mechanism for the generated schema.
type _TypeKey int

// An array of inputs used as typed arguments and converted from their type into
// function objects that are dynamically constructed and executed.
type _ValidatorInputs []interface{}

// _ValidateDurationMin is the minimum duration to accept as input
type _ValidateDurationMin string

// _ValidateIntMin is the minimum integer value to accept as input
type _ValidateIntMin int

// _ValidateRegexp is a regexp pattern to use to validate schema input.
type _ValidateRegexp string

const (
	// _SourceUserRequired indicates the parameter must be provided by the user in
	// their configuration.
	_SourceUserRequired _SourceFlags = 1 << iota

	// _SourceUserOptional indicates the parameter may optionally be specified by
	// the user in their configuration.
	_SourceUserOptional

	// _SourceAPIResult indicates the parameter may only be set by the return of
	// an API call.
	_SourceAPIResult

	// _SourceLocalFilter indicates the parameter is only used as input to the
	// resource or data source and not to be entered into the state file.
	_SourceLocalFilter
)

const (
	// _ModifyState is a mask that selects all attribute sources that can modify
	// the state (i.e. everything but filters used in data sources).
	_ModifyState = _SourceUserRequired | _SourceUserOptional | _SourceAPIResult

	// _ComputedAttrMask is a mask that selects _Source*'s that are Computed in the
	// schema.
	_ComputedAttrMask = _SourceAPIResult

	// _OptionalAttrMask is a mask that selects _Source*'s that are Optional in the
	// schema.
	_OptionalAttrMask = _SourceAPIResult | _SourceLocalFilter

	// _RequiredAttrMask is a mask that selects _Source*'s that are Required in the
	// schema.
	_RequiredAttrMask = _SourceUserRequired
)

type _TypeEntry struct {
	APIName       _APIAttr
	APIAliases    []_APIAttr
	Source        _SourceFlags
	Default       interface{}
	Description   string
	SchemaName    _SchemaAttr
	Type          schema.ValueType
	ValidateFuncs []interface{}
	SetMembers    map[_TypeKey]*_TypeEntry
	ListSchema    map[_TypeKey]*_TypeEntry

	// APITest, if returns true, will call APIToState.  The if the value was
	// found, the second return parameter will include the value that should be
	// set in the state store.
	APITest func(*_TypeEntry, interface{}) (interface{}, bool)

	// APIToState takes the value from APITest and writes it to the _AttrWriter
	APIToState func(*_TypeEntry, interface{}, _AttrWriter) error

	// ConfigRead, if it returns true, returned a value that will be passed to its
	// ConfigUse handler.
	ConfigRead func(*_TypeEntry, _AttrReader) (interface{}, bool)

	// ConfigUse takes the value returned from ConfigRead as the second argument
	// and a 3rd optional opaque context argument.
	ConfigUse func(e *_TypeEntry, v interface{}, target interface{}) error
}

type _TypeHandlers struct {
	APITest    func(*_TypeEntry, interface{}) (interface{}, bool)
	APIToState func(*_TypeEntry, interface{}, _AttrWriter) error
}

var _TypeHandlerLookupMap = map[schema.ValueType]*_TypeHandlers{
	schema.TypeBool: &_TypeHandlers{
		APITest:    _APITestBool,
		APIToState: _APIToStateBool,
	},
	schema.TypeFloat: &_TypeHandlers{
		APITest:    _APITestFloat64,
		APIToState: _APIToStateFloat64,
	},
	schema.TypeList: &_TypeHandlers{
		APITest:    _APITestList,
		APIToState: _APIToStateList,
	},
	schema.TypeMap: &_TypeHandlers{
		APITest:    _APITestMap,
		APIToState: _APIToStateMap,
	},
	schema.TypeSet: &_TypeHandlers{
		APITest:    _APITestSet,
		APIToState: _APIToStateSet,
	},
	schema.TypeString: &_TypeHandlers{
		APITest:    _APITestString,
		APIToState: _APIToStateString,
	},
}

func _APITestBool(e *_TypeEntry, self interface{}) (interface{}, bool) {
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

func _APITestFloat64(e *_TypeEntry, self interface{}) (interface{}, bool) {
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

func _APITestID(e *_TypeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	v, _ := _APITestString(e, m)

	// Unconditionally return true so that the call to the APIToState handler can
	// return an error.
	return v, true
}

func _APITestList(e *_TypeEntry, self interface{}) (interface{}, bool) {
	m := self.(map[string]interface{})

	names := append([]_APIAttr{e.APIName}, e.APIAliases...)
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

func _APITestMap(e *_TypeEntry, selfRaw interface{}) (interface{}, bool) {
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

func _APITestSet(e *_TypeEntry, selfRaw interface{}) (interface{}, bool) {
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

func _APITestString(e *_TypeEntry, selfRaw interface{}) (interface{}, bool) {
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

func _APIToStateBool(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	return w.SetBool(e.SchemaName, v.(bool))
}

func _APIToStateID(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	s, ok := v.(string)
	if !ok || len(s) == 0 {
		return fmt.Errorf("Unable to set %q's ID to an empty or non-string value: %#v", e.SchemaName, v)
	}

	stateWriter, ok := w.(*_AttrWriterState)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to SetID with a non-_AttrWriterState")
	}

	stateWriter.SetID(s)

	return nil
}

func _APIToStateFloat64(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	f, ok := v.(float64)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a float64", e.SchemaName)
	}

	return w.SetFloat64(e.SchemaName, f)
}

func _APIToStateList(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	l, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a list", e.SchemaName)
	}

	return w.SetList(e.SchemaName, l)
}

func _APIToStateMap(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	rawMap, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a map", e.SchemaName)
	}

	mWriter := _NewMapWriter(make(map[string]interface{}, len(rawMap)))

	// Make a lookup map by API Schema Name
	var setMembersLen int
	if e.SetMembers != nil {
		setMembersLen = len(e.SetMembers)
	}
	apiLookup := make(map[string]*_TypeEntry, setMembersLen)
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
			if err := mWriter.Set(_SchemaAttr(k), v); err != nil {
				return errwrap.Wrapf("Unable to store map in state: {{err}}", err)
			}
		}
	}

	return w.SetMap(e.SchemaName, mWriter.ToMap())
}

func _APIToStateSet(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	s, ok := v.([]map[string]interface{})
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a set", e.SchemaName)
	}

	set := schema.NewSet(schema.HashResource(nil), nil)
	set.Add(s)

	return w.SetSet(e.SchemaName, set)
}

func _APIToStateString(e *_TypeEntry, v interface{}, w _AttrWriter) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("PROVIDER BUG: unable to cast %s to a float64", e.SchemaName)
	}

	return w.SetString(e.SchemaName, s)
}

func _HashMap(in interface{}) int {
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
	const _DefaultHashBufSize = 4096
	b.Grow(_DefaultHashBufSize)

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

func _Indirect(v interface{}) interface{} {
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

func (e *_TypeEntry) LookupDefaultTypeHandler() *_TypeHandlers {
	h, found := _TypeHandlerLookupMap[e.Type]
	if !found {
		panic(fmt.Sprintf("PROVIDER BUG: unable to lookup %q's type (%#v)", e.SchemaName, e.Type))
	}

	return h
}

func (e *_TypeEntry) MustLookupTypeHandler() *_TypeHandlers {
	h := &_TypeHandlers{
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

// _NegateBoolToState is a factory function that creates a new function that
// negates whatever the bool is that's passed in as an argument.
func _NegateBoolToState(fn func(*_TypeEntry, interface{}, _AttrWriter) error) func(*_TypeEntry, interface{}, _AttrWriter) error {
	return func(e *_TypeEntry, v interface{}, w _AttrWriter) error {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("Unable to type assert non-bool value: %#v", v)
		}

		return fn(e, !b, w)
	}
}

// _StateSet sets an attribute based on an attrName.  Return an error if the
// Set() to schema.ResourceData fails.
func _StateSet(d *schema.ResourceData, attrName _SchemaAttr, v interface{}) error {
	if err := d.Set(string(attrName), _Indirect(v)); err != nil {
		return fmt.Errorf("PROVIDER BUG: failed set schema attribute %s to value %#v: %v", attrName, v, err)
	}

	return nil
}

func _TypeEntryListToSchema(e *_TypeEntry) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		string(e.SchemaName): e.ToSchema(),
	}
}

func _TypeEntryMapToResource(in map[_TypeKey]*_TypeEntry) *schema.Resource {
	return &schema.Resource{
		Schema: _TypeEntryMapToSchema(in),
	}
}

func _TypeEntryMapToSchema(in map[_TypeKey]*_TypeEntry) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(in))
	for _, e := range in {
		out[string(e.SchemaName)] = e.ToSchema()
	}

	return out
}

func (e *_TypeEntry) Validate() {
	if e.Source&_SourceAPIResult != 0 && e.Type == schema.TypeSet {
		panic(fmt.Sprintf("PROVIDER BUG: %s can not be computed and of type Set", e.SchemaName))
	}

	if e.Source&_SourceLocalFilter != 0 {
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

	if e.Source&(_SourceUserRequired|_SourceAPIResult) == (_SourceUserRequired | _SourceAPIResult) {
		panic(fmt.Sprintf("PROVIDER BUG: %#v and %#v are mutually exclusive Source flags", _SourceUserRequired, _SourceAPIResult))
	}
}

// MakeValidateionFunc takes a list of typed validator inputs from the receiver
// and creates a validation closure that calls each validator in serial until
// either a warning or error is returned from the first validation function.
func (e *_TypeEntry) MakeValidationFunc() func(v interface{}, key string) (warnings []string, errors []error) {
	if len(e.ValidateFuncs) == 0 {
		return nil
	}

	fns := make([]func(v interface{}, key string) (warnings []string, errors []error), 0, len(e.ValidateFuncs))
	for _, v := range e.ValidateFuncs {
		switch u := v.(type) {
		case _ValidateDurationMin:
			fns = append(fns, _ValidateDurationMinFactory(e, string(u)))
		case _ValidateIntMin:
			fns = append(fns, _ValidateIntMinFactory(e, int(u)))
		case _ValidateRegexp:
			fns = append(fns, _ValidateRegexpFactory(e, string(u)))
		}
	}

	return func(v interface{}, key string) (warnings []string, errors []error) {
		for _, fn := range fns {
			warnings, errors = fn(v, key)
			if len(warnings) > 0 || len(errors) > 0 {
				break
			}
		}
		return warnings, errors
	}
}

func (e *_TypeEntry) ToSchema() *schema.Schema {
	e.Validate()

	attr := &schema.Schema{
		Computed:     e.Source&_ComputedAttrMask != 0,
		Default:      e.Default,
		Description:  e.Description,
		Optional:     e.Source&_OptionalAttrMask != 0,
		Required:     e.Source&_RequiredAttrMask != 0,
		Type:         e.Type,
		ValidateFunc: e.MakeValidationFunc(),
	}

	// Fixup the type: use the real type vs a surrogate type
	switch e.Type {
	case schema.TypeList:
		if e.ListSchema == nil {
			attr.Elem = &schema.Schema{
				Type: schema.TypeString,
			}
		} else {
			attr.Elem = _TypeEntryMapToResource(e.ListSchema)
		}

	case schema.TypeSet:
		attr.Elem = &schema.Resource{
			Schema: _TypeEntryMapToSchema(e.SetMembers),
		}
	}

	return attr
}

func _MapStringToMapInterface(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func _ValidateDurationMinFactory(e *_TypeEntry, minDuration string) func(v interface{}, key string) (warnings []string, errors []error) {
	dMin, err := time.ParseDuration(minDuration)
	if err != nil {
		panic(fmt.Sprintf("PROVIDER BUG: duration %q not valid: %#v", minDuration, err))
	}

	return func(v interface{}, key string) (warnings []string, errors []error) {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			errors = append(errors, errwrap.Wrapf(fmt.Sprintf("Invalid %s specified (%q): {{err}}", e.SchemaName), err))
		}

		if d < dMin {
			errors = append(errors, fmt.Errorf("Invalid %s specified: duration %q less than the required minimum %s", e.SchemaName, v.(string), dMin))
		}

		return warnings, errors
	}
}

func _ValidateIntMinFactory(e *_TypeEntry, min int) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(int) < min {
			errors = append(errors, fmt.Errorf("Invalid %s specified: %d less than the required minimum %d", e.SchemaName, v.(int), min))
		}

		return warnings, errors
	}
}

func _ValidateRegexpFactory(e *_TypeEntry, reString string) func(v interface{}, key string) (warnings []string, errors []error) {
	re := regexp.MustCompile(reString)

	return func(v interface{}, key string) (warnings []string, errors []error) {
		if !re.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): regexp failed to match string", e.SchemaName, v.(string)))
		}

		return warnings, errors
	}
}
