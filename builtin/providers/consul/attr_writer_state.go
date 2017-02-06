package consul

import "github.com/hashicorp/terraform/helper/schema"

type _AttrWriterState struct {
	d *schema.ResourceData
}

func _NewStateWriter(d *schema.ResourceData) *_AttrWriterState {
	return &_AttrWriterState{
		d: d,
	}
}

func (w *_AttrWriterState) BackingType() string {
	return "state"
}

func (w *_AttrWriterState) SetBool(name _SchemaAttr, b bool) error {
	return _StateSet(w.d, name, b)
}

func (w *_AttrWriterState) SetID(id string) {
	w.d.SetId(id)
}

func (w *_AttrWriterState) SetFloat64(name _SchemaAttr, f float64) error {
	return _StateSet(w.d, name, f)
}

func (w *_AttrWriterState) SetList(name _SchemaAttr, l []interface{}) error {
	return _StateSet(w.d, name, l)
}

func (w *_AttrWriterState) SetMap(name _SchemaAttr, m map[string]interface{}) error {
	return _StateSet(w.d, name, m)
}

func (w *_AttrWriterState) SetSet(name _SchemaAttr, s *schema.Set) error {
	return _StateSet(w.d, name, []interface{}{s})
}

func (w *_AttrWriterState) SetString(name _SchemaAttr, s string) error {
	return _StateSet(w.d, name, s)
}
