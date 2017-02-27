package circonus

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

type configReader struct {
	ctxt *providerContext
	d    *schema.ResourceData
}

func newConfigReader(ctxt *providerContext, d *schema.ResourceData) *configReader {
	return &configReader{
		ctxt: ctxt,
		d:    d,
	}
}

func (r *configReader) BackingType() string {
	return "config"
}

func (r *configReader) Context() *providerContext {
	return r.ctxt
}

func (r *configReader) GetBool(attrName schemaAttr) bool {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(bool)
	}

	return false
}

func (r *configReader) GetBoolOK(attrName schemaAttr) (b, ok bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(bool), true
	}

	return false, false
}

func (r *configReader) GetDurationOK(attrName schemaAttr) (time.Duration, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return time.Duration(0), false
		}
		return d, true
	}

	return time.Duration(0), false
}

func (r *configReader) GetFloat64OK(attrName schemaAttr) (float64, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(float64), true
	}

	return 0.0, false
}

func (r *configReader) GetIntOK(attrName schemaAttr) (int, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(int), true
	}

	return 0, false
}

func (r *configReader) GetIntPtr(attrName schemaAttr) *int {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		i := v.(int)
		return &i
	}

	return nil
}

func (r *configReader) GetListOK(attrName schemaAttr) (interfaceList, bool) {
	if listRaw, ok := r.d.GetOk(string(attrName)); ok {
		return interfaceList{listRaw.([]interface{})}, true
	}
	return nil, false
}

func (r *configReader) GetMap(attrName schemaAttr) interfaceMap {
	if listRaw, ok := r.d.GetOk(string(attrName)); ok {
		m := make(map[string]interface{}, len(listRaw.(map[string]interface{})))
		for k, v := range listRaw.(map[string]interface{}) {
			m[k] = v
		}
		return interfaceMap(m)
	}
	return nil
}

func (r *configReader) GetSetAsListOK(attrName schemaAttr) (interfaceList, bool) {
	if listRaw, ok := r.d.GetOk(string(attrName)); ok {
		return listRaw.(*schema.Set).List(), true
	}
	return nil, false
}

func (r *configReader) GetString(attrName schemaAttr) string {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(string)
	}

	return ""
}

func (r *configReader) GetStringOK(attrName schemaAttr) (string, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(string), true
	}

	return "", false
}

func (r *configReader) GetStringPtr(attrName schemaAttr) *string {
	if v, ok := r.d.GetOk(string(attrName)); ok {
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

func (r *configReader) GetStringSlice(attrName schemaAttr) []string {
	if listRaw, ok := r.d.GetOk(string(attrName)); ok {
		return listRaw.([]string)
	}
	return nil
}

func (r *configReader) GetTags(attrName schemaAttr) circonusTags {
	if tagsRaw, ok := r.d.GetOk(string(attrName)); ok {
		tagPtrs := flattenSet(tagsRaw.(*schema.Set))
		return injectTagPtr(r.ctxt, tagPtrs)
	}

	return injectTag(r.ctxt, circonusTags{})
}
