package consul

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"

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
)

type _TypeEntry struct {
	APIName       _APIAttr
	APIAliases    []_APIAttr
	Source        _SourceFlags
	Description   string
	SchemaName    _SchemaAttr
	Type          schema.ValueType
	ValidateFuncs []interface{}
	SetMembers    map[_TypeKey]*_TypeEntry

	// APITest, if returns true, will call APIToState.  The if the value was
	// found, the second return parameter will include the value that should be
	// set in the state store.
	APITest func(*_TypeEntry, map[string]interface{}) (interface{}, bool)

	// APIToState takes the value from APITest and writes it to the _AttrWriter
	APIToState func(*_TypeEntry, interface{}, _AttrWriter) error
}

type _TypeHandlers struct {
	APITest    func(*_TypeEntry, map[string]interface{}) (interface{}, bool)
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

func _APITestBool(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
	v, found := self[string(e.APIName)]
	if found {
		if b, ok := v.(bool); ok {
			return b, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails bool type assertion", e.SchemaName))
		}
	}

	return false, false
}

func _APITestFloat64(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
	v, found := self[string(e.APIName)]
	if found {
		if f, ok := v.(float64); ok {
			return f, true
		} else {
			panic(fmt.Sprintf("PROVIDER BUG: %q fails float64 type assertion", e.SchemaName))
		}
	}
	return 0.0, false
}

func _APITestID(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
	v, _ := _APITestString(e, self)

	// Unconditionally return true so that the call to the APIToState handler can
	// return an error.
	return v, true
}

func _APITestList(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
	names := append([]_APIAttr{e.APIName}, e.APIAliases...)
	const defaultListLen = 8
	l := make([]interface{}, 0, defaultListLen)

	var foundName bool
	for _, name := range names {
		v, found := self[string(name)]
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

func _APITestMap(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
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

func _APITestSet(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
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

func _APITestString(e *_TypeEntry, self map[string]interface{}) (interface{}, bool) {
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

func _TypeEntryMapToSchema(in map[_TypeKey]*_TypeEntry) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(in))
	for _, e := range in {
		e.Validate()

		attr := &schema.Schema{
			Type:         e.Type,
			Description:  e.Description,
			Optional:     e.Source&_SourceAPIResult == _SourceAPIResult,
			Required:     e.Source&_SourceUserRequired == _SourceUserRequired,
			Computed:     e.Source&_SourceAPIResult == _SourceAPIResult,
			ValidateFunc: e.MakeValidationFunc(),
		}

		// Fixup the type: use the real type vs a surrogate type
		switch e.Type {
		case schema.TypeList:
			attr.Elem = &schema.Schema{
				Type: schema.TypeString,
			}
		case schema.TypeSet:
			attr.Elem = &schema.Resource{
				Schema: _TypeEntryMapToSchema(e.SetMembers),
			}
		}

		out[string(e.SchemaName)] = attr
	}

	return out
}

func (e *_TypeEntry) Validate() {
	if e.Source&_SourceAPIResult == _SourceAPIResult && e.Type == schema.TypeSet {
		panic(fmt.Sprintf("PROVIDER BUG: %s can not be computed and of type Set", e.SchemaName))
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

	fns := make([]func(v interface{}, key string) (warnings []string, errors []error), len(e.ValidateFuncs))
	for _, v := range e.ValidateFuncs {
		switch u := v.(type) {
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

// _ValidateFuncs takes a list of typed validator inputs, creates validation
// functions for each and then runs them in serial until either a warning or
// error is returned from the first validation function.
func _ValidateFuncs(e *_TypeEntry, in ...interface{}) func(v interface{}, key string) (warnings []string, errors []error) {
	if len(in) == 0 {
		return nil
	}

	fns := make([]func(v interface{}, key string) (warnings []string, errors []error), len(in))
	for _, v := range in {
		switch v.(type) {
		case _ValidateRegexp:
			fns = append(fns, _ValidateRegexpFactory(e, v.(string)))
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

func _ValidateRegexpFactory(e *_TypeEntry, reString string) func(v interface{}, key string) (warnings []string, errors []error) {
	re := regexp.MustCompile(reString)

	return func(v interface{}, key string) (warnings []string, errors []error) {
		if !re.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): regexp failed to match string", e.SchemaName, v.(string)))
		}

		return warnings, errors
	}
}
