package consul

import "github.com/hashicorp/terraform/helper/schema"

type attrWriterState struct {
	d *schema.ResourceData
}

func newStateWriter(d *schema.ResourceData) *attrWriterState {
	return &attrWriterState{
		d: d,
	}
}

func (w *attrWriterState) BackingType() string {
	return "state"
}

func (w *attrWriterState) SetBool(name schemaAttr, b bool) error {
	return stateSet(w.d, name, b)
}

func (w *attrWriterState) SetID(id string) {
	w.d.SetId(id)
}

func (w *attrWriterState) SetFloat64(name schemaAttr, f float64) error {
	return stateSet(w.d, name, f)
}

func (w *attrWriterState) SetList(name schemaAttr, l []interface{}) error {
	return stateSet(w.d, name, l)
}

func (w *attrWriterState) SetMap(name schemaAttr, m map[string]interface{}) error {
	return stateSet(w.d, name, m)
}

func (w *attrWriterState) SetSet(name schemaAttr, s *schema.Set) error {
	return stateSet(w.d, name, []interface{}{s})
}

func (w *attrWriterState) SetString(name schemaAttr, s string) error {
	return stateSet(w.d, name, s)
}
