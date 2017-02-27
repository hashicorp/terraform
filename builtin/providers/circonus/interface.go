package circonus

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

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
			panic(fmt.Sprintf("PROVIDER BUG: unable to convert %#v to list", e))
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

func (m interfaceMap) GetBool(attrName schemaAttr) bool {
	if v, ok := m[string(attrName)]; ok {
		return v.(bool)
	}

	panic("PROVIDER BUG: GetBool can only be used when a default is provided in schema")
}

func (m interfaceMap) GetBoolOK(attrName schemaAttr) (b, ok bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(bool), true
	}

	return false, false
}

func (m interfaceMap) GetFloat64OK(attrName schemaAttr) (float64, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(float64), true
	}

	return 0.0, false
}

func (m interfaceMap) GetIntOK(attrName schemaAttr) (int, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(int), true
	}

	return 0, false
}

func (m interfaceMap) GetIntPtr(attrName schemaAttr) *int {
	if v, ok := m[string(attrName)]; ok {
		i := v.(int)
		return &i
	}

	return nil
}

func (m interfaceMap) GetString(attrName schemaAttr) string {
	if v, ok := m[string(attrName)]; ok {
		return v.(string)
	}

	return ""
}

func (m interfaceMap) GetStringOK(attrName schemaAttr) (string, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(string), true
	}

	return "", false
}

func (m interfaceMap) GetStringPtr(attrName schemaAttr) *string {
	if v, ok := m[string(attrName)]; ok {
		switch v.(type) {
		case string:
			s := v.(string)
			return &s
		case *string:
			return v.(*string)
		}
	}

	return nil
}

func (m interfaceMap) GetTags(ctxt *providerContext, attrName schemaAttr) circonusTags {
	if tagsRaw, ok := m[string(attrName)]; ok {
		tagList := flattenSet(tagsRaw.(*schema.Set))
		tags := make(circonusTags, 0, len(tagList))
		for i := range tagList {
			if tagList[i] == nil || *tagList[i] == "" {
				continue
			}

			tags = append(tags, circonusTag(*tagList[i]))
		}
		return injectTag(ctxt, tags)
	}

	return injectTag(ctxt, circonusTags{})
}
