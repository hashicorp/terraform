package circonus

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

type _InterfaceList []interface{}
type _InterfaceMap map[string]interface{}

// _NewInterfaceMap returns a helper type that has methods for common operations
// for accessing data.
func _NewInterfaceMap(l interface{}) _InterfaceMap {
	return _InterfaceMap(l.(map[string]interface{}))
}

// CollectList returns []string of values that matched the key attrName.
// _InterfaceList most likely came from a schema.TypeSet.
func (l _InterfaceList) CollectList(attrName _SchemaAttr) []string {
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
func (l _InterfaceList) List() []string {
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
// _InterfaceMap most likely came from a schema.TypeSet.
func (m _InterfaceMap) CollectList(attrName _SchemaAttr) []string {
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
// _InterfaceMap most likely came from a schema.TypeSet.
func (m _InterfaceMap) CollectMap(attrName _SchemaAttr) map[string]string {
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

func (m _InterfaceMap) GetBool(attrName _SchemaAttr) bool {
	if v, ok := m[string(attrName)]; ok {
		return v.(bool)
	}

	panic("PROVIDER BUG: GetBool can only be used when a default is provided in schema")
}

func (m _InterfaceMap) GetBoolOK(attrName _SchemaAttr) (b, ok bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(bool), true
	}

	return false, false
}

func (m _InterfaceMap) GetFloat64OK(attrName _SchemaAttr) (float64, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(float64), true
	}

	return 0.0, false
}

func (m _InterfaceMap) GetIntOK(attrName _SchemaAttr) (int, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(int), true
	}

	return 0, false
}

func (m _InterfaceMap) GetIntPtr(attrName _SchemaAttr) *int {
	if v, ok := m[string(attrName)]; ok {
		i := v.(int)
		return &i
	}

	return nil
}

func (m _InterfaceMap) GetString(attrName _SchemaAttr) string {
	if v, ok := m[string(attrName)]; ok {
		return v.(string)
	}

	return ""
}

func (m _InterfaceMap) GetStringOK(attrName _SchemaAttr) (string, bool) {
	if v, ok := m[string(attrName)]; ok {
		return v.(string), true
	}

	return "", false
}

func (m _InterfaceMap) GetStringPtr(attrName _SchemaAttr) *string {
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

func (m _InterfaceMap) GetTags(ctxt *_ProviderContext, attrName _SchemaAttr) _Tags {
	if tagsRaw, ok := m[string(attrName)]; ok {
		tagList := flattenSet(tagsRaw.(*schema.Set))
		tags := make(_Tags, 0, len(tagList))
		for i := range tagList {
			if tagList[i] == nil || *tagList[i] == "" {
				continue
			}

			tags = append(tags, _Tag(*tagList[i]))
		}
		return injectTag(ctxt, tags)
	}

	return injectTag(ctxt, _Tags{})
}
