package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
)

// MapFieldWriter writes data into a single map[string]string structure.
type MapFieldWriter struct {
	Schema map[string]*Schema

	lock   sync.Mutex
	result map[string]string
}

// Map returns the underlying map that is being written to.
func (w *MapFieldWriter) Map() map[string]string {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.result == nil {
		w.result = make(map[string]string)
	}

	return w.result
}

func (w *MapFieldWriter) WriteField(addr []string, value interface{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.result == nil {
		w.result = make(map[string]string)
	}

	schemaList := addrToSchema(addr, w.Schema)
	if len(schemaList) == 0 {
		return fmt.Errorf("Invalid address to set: %#v", addr)
	}

	// If we're setting anything other than a list root or set root,
	// then disallow it.
	for _, schema := range schemaList[:len(schemaList)-1] {
		if schema.Type == TypeList {
			return fmt.Errorf(
				"%s: can only set full list",
				strings.Join(addr, "."))
		}

		if schema.Type == TypeMap {
			return fmt.Errorf(
				"%s: can only set full map",
				strings.Join(addr, "."))
		}

		if schema.Type == TypeSet {
			return fmt.Errorf(
				"%s: can only set full set",
				strings.Join(addr, "."))
		}
	}

	return w.set(addr, value)
}

func (w *MapFieldWriter) set(addr []string, value interface{}) error {
	schemaList := addrToSchema(addr, w.Schema)
	if len(schemaList) == 0 {
		return fmt.Errorf("Invalid address to set: %#v", addr)
	}

	schema := schemaList[len(schemaList)-1]
	switch schema.Type {
	case TypeBool, TypeInt, TypeFloat, TypeString:
		return w.setPrimitive(addr, value, schema)
	case TypeList:
		return w.setList(addr, value, schema)
	case TypeMap:
		return w.setMap(addr, value, schema)
	case TypeSet:
		return w.setSet(addr, value, schema)
	case typeObject:
		return w.setObject(addr, value, schema)
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (w *MapFieldWriter) setList(
	addr []string,
	v interface{},
	schema *Schema) error {
	k := strings.Join(addr, ".")
	setElement := func(idx string, value interface{}) error {
		addrCopy := make([]string, len(addr), len(addr)+1)
		copy(addrCopy, addr)
		return w.set(append(addrCopy, idx), value)
	}

	var vs []interface{}
	if err := mapstructure.Decode(v, &vs); err != nil {
		return fmt.Errorf("%s: %s", k, err)
	}

	// Set the entire list.
	var err error
	for i, elem := range vs {
		is := strconv.FormatInt(int64(i), 10)
		err = setElement(is, elem)
		if err != nil {
			break
		}
	}
	if err != nil {
		for i, _ := range vs {
			is := strconv.FormatInt(int64(i), 10)
			setElement(is, nil)
		}

		return err
	}

	w.result[k+".#"] = strconv.FormatInt(int64(len(vs)), 10)
	return nil
}

func (w *MapFieldWriter) setMap(
	addr []string,
	value interface{},
	schema *Schema) error {
	k := strings.Join(addr, ".")
	v := reflect.ValueOf(value)
	vs := make(map[string]interface{})

	if value == nil {
		// The empty string here means the map is removed.
		w.result[k] = ""
		return nil
	}

	if v.Kind() != reflect.Map {
		return fmt.Errorf("%s: must be a map", k)
	}
	if v.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%s: keys must strings", k)
	}
	for _, mk := range v.MapKeys() {
		mv := v.MapIndex(mk)
		vs[mk.String()] = mv.Interface()
	}

	// Remove the pure key since we're setting the full map value
	delete(w.result, k)

	// Set each subkey
	addrCopy := make([]string, len(addr), len(addr)+1)
	copy(addrCopy, addr)
	for subKey, v := range vs {
		if err := w.set(append(addrCopy, subKey), v); err != nil {
			return err
		}
	}

	// Set the count
	w.result[k+".#"] = strconv.Itoa(len(vs))

	return nil
}

func (w *MapFieldWriter) setObject(
	addr []string,
	value interface{},
	schema *Schema) error {
	// Set the entire object. First decode into a proper structure
	var v map[string]interface{}
	if err := mapstructure.Decode(value, &v); err != nil {
		return fmt.Errorf("%s: %s", strings.Join(addr, "."), err)
	}

	// Make space for additional elements in the address
	addrCopy := make([]string, len(addr), len(addr)+1)
	copy(addrCopy, addr)

	// Set each element in turn
	var err error
	for k1, v1 := range v {
		if err = w.set(append(addrCopy, k1), v1); err != nil {
			break
		}
	}
	if err != nil {
		for k1, _ := range v {
			w.set(append(addrCopy, k1), nil)
		}
	}

	return err
}

func (w *MapFieldWriter) setPrimitive(
	addr []string,
	v interface{},
	schema *Schema) error {
	k := strings.Join(addr, ".")

	if v == nil {
		// The empty string here means the value is removed.
		w.result[k] = ""
		return nil
	}

	var set string
	switch schema.Type {
	case TypeBool:
		var b bool
		if err := mapstructure.Decode(v, &b); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}

		set = strconv.FormatBool(b)
	case TypeString:
		if err := mapstructure.Decode(v, &set); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}
	case TypeInt:
		var n int
		if err := mapstructure.Decode(v, &n); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}
		set = strconv.FormatInt(int64(n), 10)
	case TypeFloat:
		var n float64
		if err := mapstructure.Decode(v, &n); err != nil {
			return fmt.Errorf("%s: %s", k, err)
		}
		set = strconv.FormatFloat(float64(n), 'G', -1, 64)
	default:
		return fmt.Errorf("Unknown type: %#v", schema.Type)
	}

	w.result[k] = set
	return nil
}

func (w *MapFieldWriter) setSet(
	addr []string,
	value interface{},
	schema *Schema) error {
	addrCopy := make([]string, len(addr), len(addr)+1)
	copy(addrCopy, addr)
	k := strings.Join(addr, ".")

	if value == nil {
		w.result[k+".#"] = "0"
		return nil
	}

	// If it is a slice, then we have to turn it into a *Set so that
	// we get the proper order back based on the hash code.
	if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
		// Build a temp *ResourceData to use for the conversion
		tempSchema := *schema
		tempSchema.Type = TypeList
		tempSchemaMap := map[string]*Schema{addr[0]: &tempSchema}
		tempW := &MapFieldWriter{Schema: tempSchemaMap}

		// Set the entire list, this lets us get sane values out of it
		if err := tempW.WriteField(addr, value); err != nil {
			return err
		}

		// Build the set by going over the list items in order and
		// hashing them into the set. The reason we go over the list and
		// not the `value` directly is because this forces all types
		// to become []interface{} (generic) instead of []string, which
		// most hash functions are expecting.
		s := &Set{F: schema.Set}
		tempR := &MapFieldReader{
			Map:    BasicMapReader(tempW.Map()),
			Schema: tempSchemaMap,
		}
		for i := 0; i < v.Len(); i++ {
			is := strconv.FormatInt(int64(i), 10)
			result, err := tempR.ReadField(append(addrCopy, is))
			if err != nil {
				return err
			}
			if !result.Exists {
				panic("set item just set doesn't exist")
			}

			s.Add(result.Value)
		}

		value = s
	}

	for code, elem := range value.(*Set).m {
		if err := w.set(append(addrCopy, code), elem); err != nil {
			return err
		}
	}

	w.result[k+".#"] = strconv.Itoa(value.(*Set).Len())
	return nil
}
