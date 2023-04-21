package structured

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
)

// ChangeMap is a Change that represents a Map or an Object type, and has
// converted the relevant interfaces into maps for easier access.
type ChangeMap struct {
	// Before contains the value before the proposed change.
	Before map[string]interface{}

	// After contains the value after the proposed change.
	After map[string]interface{}

	// Unknown contains the unknown status of any elements/attributes of this
	// map/object.
	Unknown map[string]interface{}

	// BeforeSensitive contains the before sensitive status of any
	// elements/attributes of this map/object.
	BeforeSensitive map[string]interface{}

	// AfterSensitive contains the after sensitive status of any
	// elements/attributes of this map/object.
	AfterSensitive map[string]interface{}

	// ReplacePaths matches the same attributes in Change exactly.
	ReplacePaths attribute_path.Matcher

	// RelevantAttributes matches the same attributes in Change exactly.
	RelevantAttributes attribute_path.Matcher
}

// AsMap converts the Change into an object or map representation by converting
// the internal Before, After, Unknown, BeforeSensitive, and AfterSensitive
// data structures into generic maps.
func (change Change) AsMap() ChangeMap {
	return ChangeMap{
		Before:             genericToMap(change.Before),
		After:              genericToMap(change.After),
		Unknown:            genericToMap(change.Unknown),
		BeforeSensitive:    genericToMap(change.BeforeSensitive),
		AfterSensitive:     genericToMap(change.AfterSensitive),
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// GetChild safely packages up a Change object for the given child, handling
// all the cases where the data might be null or a static boolean.
func (m ChangeMap) GetChild(key string) Change {
	before, beforeExplicit := getFromGenericMap(m.Before, key)
	after, afterExplicit := getFromGenericMap(m.After, key)
	unknown, _ := getFromGenericMap(m.Unknown, key)
	beforeSensitive, _ := getFromGenericMap(m.BeforeSensitive, key)
	afterSensitive, _ := getFromGenericMap(m.AfterSensitive, key)

	return Change{
		BeforeExplicit:     beforeExplicit,
		AfterExplicit:      afterExplicit,
		Before:             before,
		After:              after,
		Unknown:            unknown,
		BeforeSensitive:    beforeSensitive,
		AfterSensitive:     afterSensitive,
		ReplacePaths:       m.ReplacePaths.GetChildWithKey(key),
		RelevantAttributes: m.RelevantAttributes.GetChildWithKey(key),
	}
}

// Keys returns all the possible keys for this map. The keys for the map are
// potentially hidden and spread across multiple internal data structures and
// so this function conveniently packages them up.
func (m ChangeMap) Keys() []string {
	var keys []string
	for before := range m.Before {
		keys = append(keys, before)
	}
	for after := range m.After {
		keys = append(keys, after)
	}
	for unknown := range m.Unknown {
		keys = append(keys, unknown)
	}
	for sensitive := range m.AfterSensitive {
		keys = append(keys, sensitive)
	}
	for sensitive := range m.BeforeSensitive {
		keys = append(keys, sensitive)
	}
	return keys
}

func getFromGenericMap(generic map[string]interface{}, key string) (interface{}, bool) {
	if generic == nil {
		return nil, false
	}

	if child, ok := generic[key]; ok {
		return child, ok
	}
	return nil, false
}

func genericToMap(generic interface{}) map[string]interface{} {
	if concrete, ok := generic.(map[string]interface{}); ok {
		return concrete
	}
	return nil
}
