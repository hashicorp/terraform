package differ

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
	ReplacePaths []interface{}
}

func (change Change) asMap() ChangeMap {
	return ChangeMap{
		Before:          genericToMap(change.Before),
		After:           genericToMap(change.After),
		Unknown:         genericToMap(change.Unknown),
		BeforeSensitive: genericToMap(change.BeforeSensitive),
		AfterSensitive:  genericToMap(change.AfterSensitive),
		ReplacePaths:    change.ReplacePaths,
	}
}

func (m ChangeMap) getChild(key string) Change {
	before, beforeExplicit := getFromGenericMap(m.Before, key)
	after, afterExplicit := getFromGenericMap(m.After, key)
	unknown, _ := getFromGenericMap(m.Unknown, key)
	beforeSensitive, _ := getFromGenericMap(m.BeforeSensitive, key)
	afterSensitive, _ := getFromGenericMap(m.AfterSensitive, key)

	return Change{
		BeforeExplicit:  beforeExplicit,
		AfterExplicit:   afterExplicit,
		Before:          before,
		After:           after,
		Unknown:         unknown,
		BeforeSensitive: beforeSensitive,
		AfterSensitive:  afterSensitive,
		ReplacePaths:    m.processReplacePaths(key),
	}
}

func (m ChangeMap) processReplacePaths(key string) []interface{} {
	var ret []interface{}
	for _, p := range m.ReplacePaths {
		path := p.([]interface{})

		if len(path) == 0 {
			// This means that the current value is causing a replacement but
			// not its children, so we skip as we are returning the child's
			// value.
			continue
		}

		if path[0].(string) == key {
			ret = append(ret, path[1:])
		}
	}
	return ret
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
