package circonus

// The _Metric type is the backing store of the `circonus_metric` resource.

import (
	"bytes"
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/hashcode"
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

func (m *_Metric) ParseConfig(id string, ar _AttrReader) error {
	m.ID = _MetricID(id)
	m.Name = ar.GetString(_MetricNameAttr)
	m.Status = _MetricActiveToAPIStatus(ar.GetBool(_MetricActiveAttr))
	m.Tags = tagsToAPI(ar.GetTags(_MetricTagsAttr))
	m.Type = ar.GetString(_MetricTypeAttr)
	m.Units = ar.GetStringPtr(_MetricUnitAttr)

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
	_StateSet(d, _MetricTagsAttr, tagsToState(apiToTags(m.Tags)))
	_StateSet(d, _MetricTypeAttr, m.Type)
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

func _NewMetricID() (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", errwrap.Wrapf("metric ID creation failed: {{err}}", err)
	}

	return id, nil
}

func _MetricChecksum(ar _AttrReader) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	fmt.Fprint(b, ar.GetBool(_MetricActiveAttr))
	fmt.Fprint(b, ar.GetString(_MetricNameAttr))
	tags := ar.GetTags(_MetricTagsAttr)
	for _, tag := range tags {
		fmt.Fprint(b, tag)
	}
	fmt.Fprint(b, ar.GetString(_MetricTypeAttr))
	if p := ar.GetStringPtr(_MetricUnitAttr); p != nil {
		fmt.Fprint(b, _Indirect(p))
	}

	s := b.String()
	return hashcode.String(s)
}
