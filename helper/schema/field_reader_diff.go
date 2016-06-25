package schema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// DiffFieldReader reads fields out of a diff structures.
//
// It also requires access to a Reader that reads fields from the structure
// that the diff was derived from. This is usually the state. This is required
// because a diff on its own doesn't have complete data about non-primitive
// objects such as maps, lists and sets.
//
// The Source MUST be the data that the diff was derived from. If it isn't,
// the behavior of this struct is undefined.
//
// Reading fields from a DiffFieldReader is identical to reading from
// Source except the diff will be applied to the end result.
//
// The "Exists" field on the result will be set to true if the complete
// field exists whether its from the source, diff, or a combination of both.
// It cannot be determined whether a retrieved value is composed of
// diff elements.
type DiffFieldReader struct {
	Diff   *terraform.InstanceDiff
	Source FieldReader
	Schema map[string]*Schema
}

func (r *DiffFieldReader) ReadField(address []string) (FieldReadResult, error) {
	schemaList := addrToSchema(address, r.Schema)
	if len(schemaList) == 0 {
		return FieldReadResult{}, nil
	}

	schema := schemaList[len(schemaList)-1]
	switch schema.Type {
	case TypeBool, TypeInt, TypeFloat, TypeString:
		return r.readPrimitive(address, schema)
	case TypeList:
		return r.readList(address, schema)
	case TypeMap:
		return r.readMap(address, schema)
	case TypeSet:
		return r.readSet(address, schema)
	case typeObject:
		return readObjectField(r, address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (r *DiffFieldReader) readMap(
	address []string, schema *Schema) (FieldReadResult, error) {
	result := make(map[string]interface{})
	exists := false

	// First read the map from the underlying source
	source, err := r.Source.ReadField(address)
	if err != nil {
		return FieldReadResult{}, err
	}
	if source.Exists {
		result = source.Value.(map[string]interface{})
		exists = true
	}

	// Next, read all the elements we have in our diff, and apply
	// the diff to our result.
	prefix := strings.Join(address, ".") + "."
	for k, v := range r.Diff.Attributes {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if strings.HasPrefix(k, prefix+"%") {
			// Ignore the count field
			continue
		}

		exists = true

		k = k[len(prefix):]
		if v.NewRemoved {
			delete(result, k)
			continue
		}

		result[k] = v.New
	}

	var resultVal interface{}
	if exists {
		resultVal = result
	}

	return FieldReadResult{
		Value:  resultVal,
		Exists: exists,
	}, nil
}

func (r *DiffFieldReader) readPrimitive(
	address []string, schema *Schema) (FieldReadResult, error) {
	result, err := r.Source.ReadField(address)
	if err != nil {
		return FieldReadResult{}, err
	}

	attrD, ok := r.Diff.Attributes[strings.Join(address, ".")]
	if !ok {
		return result, nil
	}

	var resultVal string
	if !attrD.NewComputed {
		resultVal = attrD.New
		if attrD.NewExtra != nil {
			result.ValueProcessed = resultVal
			if err := mapstructure.WeakDecode(attrD.NewExtra, &resultVal); err != nil {
				return FieldReadResult{}, err
			}
		}
	}

	result.Computed = attrD.NewComputed
	result.Exists = true
	result.Value, err = stringToPrimitive(resultVal, false, schema)
	if err != nil {
		return FieldReadResult{}, err
	}

	return result, nil
}

func (r *DiffFieldReader) readList(
	address []string, schema *Schema) (FieldReadResult, error) {
	prefix := strings.Join(address, ".") + "."

	addrPadded := make([]string, len(address)+1)
	copy(addrPadded, address)

	// Get the number of elements in the list
	addrPadded[len(addrPadded)-1] = "#"
	countResult, err := r.readPrimitive(addrPadded, &Schema{Type: TypeInt})
	if err != nil {
		return FieldReadResult{}, err
	}
	if !countResult.Exists {
		// No count, means we have no list
		countResult.Value = 0
	}
	// If we have an empty list, then return an empty list
	if countResult.Computed || countResult.Value.(int) == 0 {
		return FieldReadResult{
			Value:    []interface{}{},
			Exists:   countResult.Exists,
			Computed: countResult.Computed,
		}, nil
	}

	// Bail out if diff doesn't contain the given field at all
	// This has to be a separate loop because we're only
	// iterating over raw list items (list.idx).
	// Other fields (list.idx.*) are left for other read* methods
	// which can deal with these fields appropriately.
	diffContainsField := false
	for k, _ := range r.Diff.Attributes {
		if strings.HasPrefix(k, address[0]+".") {
			diffContainsField = true
		}
	}
	if !diffContainsField {
		return FieldReadResult{
			Value:  []interface{}{},
			Exists: false,
		}, nil
	}

	// Create the list that will be our result
	list := []interface{}{}

	// Go through the diff and find all the list items
	// We are not iterating over the diff directly as some indexes
	// may be missing and we expect the whole list to be returned.
	for i := 0; i < countResult.Value.(int); i++ {
		idx := strconv.Itoa(i)
		addrString := prefix + idx

		d, ok := r.Diff.Attributes[addrString]
		if ok && d.NewRemoved {
			// If the field is being removed, we ignore it
			continue
		}

		addrPadded[len(addrPadded)-1] = idx
		raw, err := r.ReadField(addrPadded)
		if err != nil {
			return FieldReadResult{}, err
		}
		if !raw.Exists {
			// This should never happen, because by the time the data
			// gets to the FieldReaders, all the defaults should be set by
			// Schema.
			panic("missing field in set: " + addrString + "." + idx)
		}
		list = append(list, raw.Value)
	}

	// Determine if the list "exists". It exists if there are items or if
	// the diff explicitly wanted it empty.
	exists := len(list) > 0
	if !exists {
		// We could check if the diff value is "0" here but I think the
		// existence of "#" on its own is enough to show it existed. This
		// protects us in the future from the zero value changing from
		// "0" to "" breaking us (if that were to happen).
		if _, ok := r.Diff.Attributes[prefix+"#"]; ok {
			exists = true
		}
	}

	return FieldReadResult{
		Value:  list,
		Exists: exists,
	}, nil
}

func (r *DiffFieldReader) readSet(
	address []string, schema *Schema) (FieldReadResult, error) {
	prefix := strings.Join(address, ".") + "."

	// Create the set that will be our result
	set := schema.ZeroValue().(*Set)

	// Check if we're supposed to remove it
	v, ok := r.Diff.Attributes[prefix+"#"]
	if ok && v.New == "0" {
		// I'm not entirely sure what's the point of
		// returning empty set w/ Exists: true
		return FieldReadResult{
			Value:  set,
			Exists: true,
		}, nil
	}

	// Compose list of all keys (diff + source)
	var keys []string

	// Add keys from diff
	diffContainsField := false
	for k, _ := range r.Diff.Attributes {
		if strings.HasPrefix(k, address[0]+".") {
			diffContainsField = true
		}
		keys = append(keys, k)
	}
	// Bail out if diff doesn't contain the given field at all
	if !diffContainsField {
		return FieldReadResult{
			Value:  set,
			Exists: false,
		}, nil
	}
	// Add keys from source
	sourceResult, err := r.Source.ReadField(address)
	if err == nil && sourceResult.Exists {
		sourceSet := sourceResult.Value.(*Set)
		sourceMap := sourceSet.Map()

		for k, _ := range sourceMap {
			key := prefix + k
			_, ok := r.Diff.Attributes[key]
			if !ok {
				keys = append(keys, key)
			}
		}
	}

	// Keep the order consistent for hashing functions
	sort.Strings(keys)

	// Go through the map and find all the set items
	// We are not iterating over the diff directly as some indexes
	// may be missing and we expect the whole set to be returned.
	for _, k := range keys {
		d, ok := r.Diff.Attributes[k]
		if ok && d.NewRemoved {
			// If the field is being removed, we ignore it
			continue
		}
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if strings.HasSuffix(k, "#") {
			// Ignore any count field
			continue
		}

		// Split the key, since it might be a sub-object like "idx.field"
		parts := strings.Split(k[len(prefix):], ".")
		idx := parts[0]

		raw, err := r.ReadField(append(address, idx))
		if err != nil {
			return FieldReadResult{}, err
		}
		if !raw.Exists {
			// This shouldn't happen because we just verified it does exist
			panic("missing field in set: " + k + "." + idx)
		}

		set.Add(raw.Value)
	}

	// Determine if the set "exists". It exists if there are items or if
	// the diff explicitly wanted it empty.
	exists := set.Len() > 0
	if !exists {
		// We could check if the diff value is "0" here but I think the
		// existence of "#" on its own is enough to show it existed. This
		// protects us in the future from the zero value changing from
		// "0" to "" breaking us (if that were to happen).
		if _, ok := r.Diff.Attributes[prefix+"#"]; ok {
			exists = true
		}
	}

	return FieldReadResult{
		Value:  set,
		Exists: exists,
	}, nil
}
