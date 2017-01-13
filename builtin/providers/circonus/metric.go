package circonus

// The _Metric type is the backing store of the `circonus_metric` resource.

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/schema"
)

type _Metric struct {
	ID _MetricID
	api.CheckBundleMetric
}

func _NewMetric() _Metric {
	return _Metric{}
}

func (m *_Metric) Create(d *schema.ResourceData) error {
	return m.SaveState(d)
}

func (m *_Metric) ParseConfig(id string, d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	m.ID = _MetricID(id)
	m.Name = _ConfigGetString(d, _MetricNameAttr)
	m.Status = _MetricActiveToAPIStatus(_ConfigGetBool(d, _MetricActiveAttr))
	m.Tags = tagsToState(_ConfigGetTags(ctxt, d, _MetricTagsAttr))
	m.Type = _ConfigGetString(d, _MetricTypeAttr)
	m.Units = _ConfigGetStringPtr(d, _MetricUnitAttr)

	return nil
}

func (m *_Metric) SaveState(d *schema.ResourceData) error {
	var active bool
	switch m.Status {
	case _MetricStatusActive:
		active = true
	case _MetricStatusAvailable:
		active = false
	default:
		panic(fmt.Sprintf("Provider bug: unsupported active type: %s", m.Status))
	}

	_StateSet(d, _MetricActiveAttr, active)
	_StateSet(d, _MetricNameAttr, m.Name)
	_StateSet(d, _MetricTagsAttr, m.Tags)
	_StateSet(d, _MetricUnitAttr, m.Units)

	d.SetId(string(m.ID))

	return nil
}

func (m *_Metric) Update(d *schema.ResourceData) error {
	// NOTE: there are no "updates" to be made against an API server, so we just
	// pass through a call to SaveState.  Keep this method around for API
	// symmetry.
	return m.SaveState(d)
}

func _MetricAPIStatusToBool(s string) bool {
	switch s {
	case _MetricStatusActive:
		return true
	case _MetricStatusAvailable:
		return false
	default:
		panic(fmt.Sprintf("PROVIDER BUG: metric status %q unsupported", s))
	}
}

func _MetricActiveToAPIStatus(active bool) string {
	switch active {
	case true:
		return _MetricStatusActive
	case false:
		return _MetricStatusAvailable
	}

	panic("suppress Go error message")
}
