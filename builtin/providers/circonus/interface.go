package circonus

type _InterfaceList []interface{}
type _InterfaceMap map[string]interface{}

// _NewInterfaceMap returns a helper type that has methods for common operations
// for accessing data.
func _NewInterfaceMap(l interface{}) _InterfaceMap {
	return _InterfaceMap(l.(map[string]interface{}))
}

// CollectKey returns []string of values that matched the key attrName.
// _InterfaceList most likely came from a schema.TypeSet.
func (l _InterfaceList) CollectKey(attrName _SchemaAttr) []string {
	stringList := make([]string, 0, len(l))

	for _, mapRaw := range l {
		mapAttrs := mapRaw.(map[string]interface{})

		if v, ok := mapAttrs[string(attrName)]; ok {
			stringList = append(stringList, v.(string))
		}
	}

	return stringList
}

func (m _InterfaceMap) GetString(attrName _SchemaAttr) string {
	if v, ok := m[string(attrName)]; ok {
		return v.(string)
	}

	return ""
}

func (m _InterfaceMap) GetStringPtr(attrName _SchemaAttr) *string {
	if v, ok := m[string(attrName)]; ok {
		s := v.(string)
		return &s
	}

	return nil
}

func (m _InterfaceMap) GetTags(ctxt *providerContext, attrName _SchemaAttr, defaultTag _Tag) _Tags {
	if tagsRaw, ok := m[string(attrName)]; ok {
		tagList := flattenSet(tagsRaw.(*schema.Set))
		tags := make(_Tags, 0, len(tagList))
		for i := range tagList {
			if tagList[i] == nil || *tagList[i] == "" {
				continue
			}

			tags = append(tags, _Tag(*tagList[i]))
		}
		return injectTag(ctxt, tags, defaultTag)
	}

	return injectTag(ctxt, _Tags{}, defaultTag)
}
