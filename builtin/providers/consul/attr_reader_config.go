package consul

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

type _ConfigReader struct {
	d *schema.ResourceData
}

func _NewConfigReader(d *schema.ResourceData) *_ConfigReader {
	return &_ConfigReader{
		d: d,
	}
}

func (r *_ConfigReader) BackingType() string {
	return "config"
}

func (r *_ConfigReader) GetBool(attrName _SchemaAttr) bool {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(bool)
	}

	return false
}

func (r *_ConfigReader) GetBoolOK(attrName _SchemaAttr) (b, ok bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(bool), true
	}

	return false, false
}

func (r *_ConfigReader) GetDurationOK(attrName _SchemaAttr) (time.Duration, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return time.Duration(0), false
		}
		return d, true
	}

	return time.Duration(0), false
}

func (r *_ConfigReader) GetFloat64OK(attrName _SchemaAttr) (float64, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(float64), true
	}

	return 0.0, false
}

func (r *_ConfigReader) GetIntOK(attrName _SchemaAttr) (int, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(int), true
	}

	return 0, false
}

func (r *_ConfigReader) GetIntPtr(attrName _SchemaAttr) *int {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		i := v.(int)
		return &i
	}

	return nil
}

func (r *_ConfigReader) GetString(attrName _SchemaAttr) string {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(string)
	}

	return ""
}

func (r *_ConfigReader) GetStringOK(attrName _SchemaAttr) (string, bool) {
	if v, ok := r.d.GetOk(string(attrName)); ok {
		return v.(string), true
	}

	return "", false
}

func (r *_ConfigReader) GetStringPtr(attrName _SchemaAttr) *string {
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

func (r *_ConfigReader) GetStringSlice(attrName _SchemaAttr) []string {
	if listRaw, ok := r.d.GetOk(string(attrName)); ok {
		return listRaw.([]string)
	}
	return nil
}
