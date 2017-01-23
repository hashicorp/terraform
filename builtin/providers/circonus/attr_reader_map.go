package circonus

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

type _MapReader struct {
	ctxt *_ProviderContext
	m    _InterfaceMap
}

func _NewMapReader(ctxt *_ProviderContext, m _InterfaceMap) *_MapReader {
	return &_MapReader{
		ctxt: ctxt,
		m:    m,
	}
}

func (r *_MapReader) BackingType() string {
	return "interface_map"
}

func (r *_MapReader) Context() *_ProviderContext {
	return r.ctxt
}

func (r *_MapReader) GetBool(attrName _SchemaAttr) bool {
	if b, ok := r.m.GetBoolOK(attrName); ok {
		return b
	}

	return false
}

func (r *_MapReader) GetBoolOK(attrName _SchemaAttr) (b, ok bool) {
	return r.m.GetBoolOK(attrName)
}

func (r *_MapReader) GetDurationOK(attrName _SchemaAttr) (time.Duration, bool) {
	if v, ok := r.m[string(attrName)]; ok {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return time.Duration(0), false
		}
		return d, true
	}

	return time.Duration(0), false
}

func (r *_MapReader) GetFloat64OK(attrName _SchemaAttr) (float64, bool) {
	if f, ok := r.m.GetFloat64OK(attrName); ok {
		return f, true
	}

	return 0.0, false
}

func (r *_MapReader) GetIntOK(attrName _SchemaAttr) (int, bool) {
	if i, ok := r.m.GetIntOK(attrName); ok {
		return i, true
	}

	return 0, false
}

func (r *_MapReader) GetListOK(attrName _SchemaAttr) (_InterfaceList, bool) {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return _InterfaceList{listRaw.([]interface{})}, true
	}
	return nil, false
}

func (r *_MapReader) GetMap(attrName _SchemaAttr) _InterfaceMap {
	if listRaw, ok := r.m[string(attrName)]; ok {
		m := make(map[string]interface{}, len(listRaw.(map[string]interface{})))
		for k, v := range listRaw.(map[string]interface{}) {
			m[k] = v
		}
		return _InterfaceMap(m)
	}
	return nil
}

func (r *_MapReader) GetSetAsListOK(attrName _SchemaAttr) (_InterfaceList, bool) {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return listRaw.(*schema.Set).List(), true
	}
	return nil, false
}

func (r *_MapReader) GetString(attrName _SchemaAttr) string {
	if s, ok := r.m.GetStringOK(attrName); ok {
		return s
	}

	return ""
}

func (r *_MapReader) GetStringPtr(attrName _SchemaAttr) *string {
	return r.m.GetStringPtr(attrName)
}

func (r *_MapReader) GetStringOK(attrName _SchemaAttr) (string, bool) {
	if s, ok := r.m.GetStringOK(attrName); ok {
		return s, true
	}

	return "", false
}

func (r *_MapReader) GetStringSlice(attrName _SchemaAttr) []string {
	if listRaw, ok := r.m[string(attrName)]; ok {
		return listRaw.([]string)
	}
	return nil
}

func (r *_MapReader) GetTags(attrName _SchemaAttr) _Tags {
	if tagsRaw, ok := r.m[string(attrName)]; ok {
		tagPtrs := flattenSet(tagsRaw.(*schema.Set))
		return injectTagPtr(r.ctxt, tagPtrs)
	}

	return injectTag(r.ctxt, _Tags{})
}
