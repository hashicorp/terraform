package circonus

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

type mapReader struct {
	ctxt *providerContext
	m    interfaceMap
}

func newMapReader(ctxt *providerContext, m interfaceMap) *mapReader {
	return &mapReader{
		ctxt: ctxt,
		m:    m,
	}
}

func (r *mapReader) BackingType() string {
	return "interface_map"
}

func (r *mapReader) Context() *providerContext {
	return r.ctxt
}

func (r *mapReader) GetBool(attrName schemaAttr) bool {
	if b, ok := r.m.GetBoolOK(attrName); ok {
		return b
	}

	return false
}

func (r *mapReader) GetBoolOK(attrName schemaAttr) (b, ok bool) {
	return r.m.GetBoolOK(attrName)
}

func (r *mapReader) GetDurationOK(attrName schemaAttr) (time.Duration, bool) {
	if v, ok := r.m[string(attrName)]; ok {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return time.Duration(0), false
		}
		return d, true
	}

	return time.Duration(0), false
}

func (r *mapReader) GetFloat64OK(attrName schemaAttr) (float64, bool) {
	if f, ok := r.m.GetFloat64OK(attrName); ok {
		return f, true
	}

	return 0.0, false
}

func (r *mapReader) GetIntOK(attrName schemaAttr) (int, bool) {
	if i, ok := r.m.GetIntOK(attrName); ok {
		return i, true
	}

	return 0, false
}

func (r *mapReader) GetIntPtr(attrName schemaAttr) *int {
	return r.m.GetIntPtr(attrName)
}

func (r *mapReader) GetListOK(attrName schemaAttr) (interfaceList, bool) {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return interfaceList{listRaw.([]interface{})}, true
	}
	return nil, false
}

func (r *mapReader) GetMap(attrName schemaAttr) interfaceMap {
	if listRaw, ok := r.m[string(attrName)]; ok {
		m := make(map[string]interface{}, len(listRaw.(map[string]interface{})))
		for k, v := range listRaw.(map[string]interface{}) {
			m[k] = v
		}
		return interfaceMap(m)
	}
	return nil
}

func (r *mapReader) GetSetAsListOK(attrName schemaAttr) (interfaceList, bool) {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return listRaw.(*schema.Set).List(), true
	}
	return nil, false
}

func (r *mapReader) GetString(attrName schemaAttr) string {
	if s, ok := r.m.GetStringOK(attrName); ok {
		return s
	}

	return ""
}

func (r *mapReader) GetStringPtr(attrName schemaAttr) *string {
	return r.m.GetStringPtr(attrName)
}

func (r *mapReader) GetStringOK(attrName schemaAttr) (string, bool) {
	if s, ok := r.m.GetStringOK(attrName); ok {
		return s, true
	}

	return "", false
}

func (r *mapReader) GetStringSlice(attrName schemaAttr) []string {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return listRaw.([]string)
	}
	return nil
}

func (r *mapReader) GetTags(attrName schemaAttr) circonusTags {
	if tagsRaw, ok := r.m[string(attrName)]; ok {
		tagPtrs := flattenSet(tagsRaw.(*schema.Set))
		return injectTagPtr(r.ctxt, tagPtrs)
	}

	return injectTag(r.ctxt, circonusTags{})
}
