package circonus

import "log"

type interfaceList []interface{}
type interfaceMap map[string]interface{}

// newInterfaceMap returns a helper type that has methods for common operations
// for accessing data.
func newInterfaceMap(l interface{}) interfaceMap {
	return interfaceMap(l.(map[string]interface{}))
}

// CollectList returns []string of values that matched the key attrName.
// interfaceList most likely came from a schema.TypeSet.
func (l interfaceList) CollectList(attrName schemaAttr) []string {
	stringList := make([]string, 0, len(l))

	for _, mapRaw := range l {
		mapAttrs := mapRaw.(map[string]interface{})

		if v, ok := mapAttrs[string(attrName)]; ok {
			stringList = append(stringList, v.(string))
		}
	}

	return stringList
}

// List returns a list of values in a Set as a string slice
func (l interfaceList) List() []string {
	stringList := make([]string, 0, len(l))
	for _, e := range l {
		switch e.(type) {
		case string:
			stringList = append(stringList, e.(string))
		case []interface{}:
			for _, v := range e.([]interface{}) {
				stringList = append(stringList, v.(string))
			}
		default:
			log.Printf("[ERROR] PROVIDER BUG: unable to convert %#v to list", e)
			return nil
		}
	}
	return stringList
}

// CollectList returns []string of values that matched the key attrName.
// interfaceMap most likely came from a schema.TypeSet.
func (m interfaceMap) CollectList(attrName schemaAttr) []string {
	stringList := make([]string, 0, len(m))

	for _, mapRaw := range m {
		mapAttrs := mapRaw.(map[string]interface{})

		if v, ok := mapAttrs[string(attrName)]; ok {
			stringList = append(stringList, v.(string))
		}
	}

	return stringList
}

// CollectMap returns map[string]string of values that matched the key attrName.
// interfaceMap most likely came from a schema.TypeSet.
func (m interfaceMap) CollectMap(attrName schemaAttr) map[string]string {
	var mergedMap map[string]string

	if attrRaw, ok := m[string(attrName)]; ok {
		attrMap := attrRaw.(map[string]interface{})
		mergedMap = make(map[string]string, len(m))
		for k, v := range attrMap {
			mergedMap[k] = v.(string)
		}
	}

	if len(mergedMap) == 0 {
		return nil
	}

	return mergedMap
}
