package consul

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

type _AttrWriterMap struct {
	m *map[string]interface{}
}

func _NewMapWriter(m *map[string]interface{}) *_AttrWriterMap {
	return &_AttrWriterMap{
		m: m,
	}
}

func (w *_AttrWriterMap) BackingType() string {
	return "map"
}

func (w *_AttrWriterMap) Set(name _SchemaAttr, v interface{}) error {
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

func (w *_AttrWriterMap) SetBool(name _SchemaAttr, b bool) error {
	(*w.m)[string(name)] = fmt.Sprintf("%t", b)
	return nil
}

func (w *_AttrWriterMap) SetFloat64(name _SchemaAttr, f float64) error {
	(*w.m)[string(name)] = strconv.FormatFloat(f, 'g', -1, 64)
	return nil
}

func (w *_AttrWriterMap) SetList(name _SchemaAttr, l []interface{}) error {
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a list within a map for %s", name))
	out := make([]string, 0, len(l))
	for i, v := range l {
		switch u := v.(type) {
		case string:
			out[i] = u
		default:
			panic(fmt.Sprintf("PROVIDER BUG: SetList type %T not supported (%#v)", v, v))
		}
	}

	(*w.m)[string(name)] = strings.Join(out, ", ")
	return nil
}

func (w *_AttrWriterMap) SetMap(name _SchemaAttr, m map[string]interface{}) error {
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a map within a map for %s", name))
}

func (w *_AttrWriterMap) SetSet(name _SchemaAttr, s *schema.Set) error {
	panic(fmt.Sprintf("PROVIDER BUG: Cat set a set within a map for %s", name))
}

func (w *_AttrWriterMap) SetString(name _SchemaAttr, s string) error {
	(*w.m)[string(name)] = s
	return nil
}
