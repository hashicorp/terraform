package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// DiffFieldReader reads fields out of a diff structures.
//
// It also requires access to a Reader that reads fields from the structure
// that the diff was derived from. This is usually the state. This is required
// because a diff on its own doesn't have complete data about full objects
// such as maps.
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

	// cache for memoizing ReadField calls.
	cache map[string]cachedFieldReadResult
}

type cachedFieldReadResult struct {
	val FieldReadResult
	err error
}

func (r *DiffFieldReader) ReadField(address []string) (FieldReadResult, error) {
	if r.cache == nil {
		r.cache = make(map[string]cachedFieldReadResult)
	}

	// Create the cache key by joining around a value that isn't a valid part
	// of an address. This assumes that the Source and Schema are not changed
	// for the life of this DiffFieldReader.
	cacheKey := strings.Join(address, "|")
	if cached, ok := r.cache[cacheKey]; ok {
		return cached.val, cached.err
	}

	schemaList := addrToSchema(address, r.Schema)
	if len(schemaList) == 0 {
		r.cache[cacheKey] = cachedFieldReadResult{}
		return FieldReadResult{}, nil
	}

	var res FieldReadResult
	var err error

	schema := schemaList[len(schemaList)-1]
	switch schema.Type {
	case TypeBool, TypeInt, TypeFloat, TypeString:
		res, err = r.readPrimitive(address, schema)
	case TypeList:
		res, err = readListField(r, address, schema)
	case TypeMap:
		res, err = r.readMap(address, schema)
	case TypeSet:
		res, err = r.readSet(address, schema)
	case typeObject:
		res, err = readObjectField(r, address, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}

	r.cache[cacheKey] = cachedFieldReadResult{
		val: res,
		err: err,
	}
	return res, err
}

func (r *DiffFieldReader) readMap(
	address []string, schema *Schema) (FieldReadResult, error) {
	result := make(map[string]interface{})
	resultSet := false

	// First read the map from the underlying source
	source, err := r.Source.ReadField(address)
	if err != nil {
		return FieldReadResult{}, err
	}
	if source.Exists {
		// readMap may return a nil value, or an unknown value placeholder in
		// some cases, causing the type assertion to panic if we don't assign the ok value
		result, _ = source.Value.(map[string]interface{})
		resultSet = true
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

		resultSet = true

		k = k[len(prefix):]
		if v.NewRemoved {
			delete(result, k)
			continue
		}

		result[k] = v.New
	}

	key := address[len(address)-1]
	err = mapValuesToPrimitive(key, result, schema)
	if err != nil {
		return FieldReadResult{}, nil
	}

	var resultVal interface{}
	if resultSet {
		resultVal = result
	}

	return FieldReadResult{
		Value:  resultVal,
		Exists: resultSet,
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

func (r *DiffFieldReader) readSet(
	address []string, schema *Schema) (FieldReadResult, error) {
	// copy address to ensure we don't modify the argument
	address = append([]string(nil), address...)

	prefix := strings.Join(address, ".") + "."

	// Create the set that will be our result
	set := schema.ZeroValue().(*Set)

	// Go through the map and find all the set items
	for k, d := range r.Diff.Attributes {
		if d.NewRemoved {
			// If the field is removed, we always ignore it
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

	if !exists {
		result, err := r.Source.ReadField(address)
		if err != nil {
			return FieldReadResult{}, err
		}
		if result.Exists {
			return result, nil
		}
	}

	return FieldReadResult{
		Value:  set,
		Exists: exists,
	}, nil
}
