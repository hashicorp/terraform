package consul

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

type attrWriterMap struct {
	m map[string]interface{}
}

func newMapWriter(m map[string]interface{}) *attrWriterMap {
	return &attrWriterMap{
		m: m,
	}
}

func (w *attrWriterMap) BackingType() string {
	return "map"
}

func (w *attrWriterMap) Set(name schemaAttr, v interface{}) error {
	switch u := v.(type) {
	case string:
		return w.SetString(name, u)
	case float64:
		return w.SetFloat64(name, u)
	case bool:
		return w.SetBool(name, u)
	case nil:
		return w.SetString(name, "")
	default:
		panic(fmt.Sprintf("PROVIDER BUG: Set type %T not supported (%#v) for %s ", v, v, name))
	}
}

func (w *attrWriterMap) SetBool(name schemaAttr, b bool) error {
	w.m[string(name)] = fmt.Sprintf("%t", b)
	return nil
}

func (w *attrWriterMap) SetFloat64(name schemaAttr, f float64) error {
	w.m[string(name)] = strconv.FormatFloat(f, 'g', -1, 64)
	return nil
}

func (w *attrWriterMap) SetList(name schemaAttr, l []interface{}) error {
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a list within a map for %s", name))
}

func (w *attrWriterMap) SetMap(name schemaAttr, m map[string]interface{}) error {
	w.m[string(name)] = m
	return nil
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a map within a map for %s", name))
}

func (w *attrWriterMap) SetSet(name schemaAttr, s *schema.Set) error {
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a set within a map for %s", name))
}

func (w *attrWriterMap) SetString(name schemaAttr, s string) error {
	w.m[string(name)] = s
	return nil
}

func (w *attrWriterMap) ToMap() map[string]interface{} {
	return w.m
}
