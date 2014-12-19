package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// DiffFieldReader reads fields out of a diff structures.
type DiffFieldReader struct {
	Diff *terraform.InstanceDiff
}

func (r *DiffFieldReader) ReadField(
	address []string, schema *Schema) (FieldReadResult, error) {
	k := strings.Join(address, ".")

	switch schema.Type {
	case TypeBool:
		fallthrough
	case TypeInt:
		fallthrough
	case TypeString:
		return r.readPrimitive(k, schema)
	case TypeList:
		return readListField(r, k, schema)
	case TypeMap:
		return r.readMap(k)
	case TypeSet:
		return r.readSet(k, schema)
	case typeObject:
		return readObjectField(r, k, schema.Elem.(map[string]*Schema))
	default:
		panic(fmt.Sprintf("Unknown type: %#v", schema.Type))
	}
}

func (r *DiffFieldReader) readMap(k string) (FieldReadResult, error) {
	result := make(map[string]interface{})
	negresult := make(map[string]interface{})
	resultSet := false

	prefix := k + "."
	for k, v := range r.Diff.Attributes {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		resultSet = true

		k = k[len(prefix):]
		if v.NewRemoved {
			negresult[k] = ""
			continue
		}

		result[k] = v.New
	}

	var resultVal interface{}
	if resultSet {
		resultVal = result
	}

	return FieldReadResult{
		Value:    resultVal,
		NegValue: negresult,
		Exists:   resultSet,
	}, nil
}

func (r *DiffFieldReader) readPrimitive(
	k string, schema *Schema) (FieldReadResult, error) {
	attrD, ok := r.Diff.Attributes[k]
	if !ok {
		return FieldReadResult{}, nil
	}
	if attrD.NewComputed {
		return FieldReadResult{
			Exists:   true,
			Computed: true,
		}, nil
	}

	result := attrD.New
	if attrD.NewExtra != nil {
		if err := mapstructure.WeakDecode(attrD.NewExtra, &result); err != nil {
			return FieldReadResult{}, err
		}
	}

	returnVal, err := stringToPrimitive(result, false, schema)
	if err != nil {
		return FieldReadResult{}, err
	}

	return FieldReadResult{
		Value:  returnVal,
		Exists: true,
	}, nil
}

func (r *DiffFieldReader) readSet(
	k string, schema *Schema) (FieldReadResult, error) {
	// Create the set that will be our result
	set := &Set{F: schema.Set}

	// Get the schema for the elements
	var elemSchema *Schema
	switch t := schema.Elem.(type) {
	case *Resource:
		elemSchema = &Schema{
			Type: typeObject,
			Elem: t.Schema,
		}
	case *Schema:
		elemSchema = t
	}

	// Go through the map and find all the set items
	prefix := k + "."
	for k, _ := range r.Diff.Attributes {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if strings.HasPrefix(k, prefix+"#") {
			// Ignore the count field
			continue
		}

		// Split the key, since it might be a sub-object like "idx.field"
		parts := strings.Split(k[len(prefix):], ".")
		idx := parts[0]

		raw, err := r.ReadField([]string{prefix + idx}, elemSchema)
		if err != nil {
			return FieldReadResult{}, err
		}
		if !raw.Exists {
			// This shouldn't happen because we just verified it does exist
			panic("missing field in set: " + k + "." + idx)
		}

		set.Add(raw.Value)
	}

	return FieldReadResult{
		Value:  set,
		Exists: true,
	}, nil
}
