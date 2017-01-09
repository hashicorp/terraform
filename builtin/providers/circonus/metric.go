package circonus

// The _Metric type is the backing store of the `circonus_metric` resource.

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
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
	return m.Save(d)
}

func (m *_Metric) ParseSchema(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	id := schemaGetString(d, _MetricIDAttr)
	if id == "" {
		var err error
		id, err = uuid.GenerateUUID()
		if err != nil {
			return errwrap.Wrapf("metric ID creation failed: {{err}}", err)
		}
	}

	var metricStatus string = _MetricStatusAvailable
	if b, ok := schemaGetBoolOK(d, _MetricActiveAttr); ok && b {
		metricStatus = _MetricStatusActive
	}

	m.ID = _MetricID(id)
	m.Name = schemaGetString(d, _MetricNameAttr)
	m.Status = metricStatus
	m.SetTags(schemaGetTags(ctxt, d, _MetricTagsAttr, _Tag{}))
	m.Type = schemaGetString(d, _MetricTypeAttr)
	m.Units = schemaGetStringPtr(d, _MetricUnitAttr)

	return nil
}

func (m *_Metric) Save(d *schema.ResourceData) error {
	var active bool
	switch m.Status {
	case _MetricStatusActive:
		active = true
	case _MetricStatusAvailable:
		active = false
	default:
		panic(fmt.Sprintf("Provider bug: unsupported active type: %s", m.Status))
	}

	stateSet(d, _MetricActiveAttr, active)
	stateSet(d, _MetricNameAttr, m.Name)
	stateSet(d, _MetricTagsAttr, apiToTags(m.Tags))
	stateSet(d, _MetricUnitAttr, m.Units)

	d.SetId(string(m.ID))

	return nil
}

func (m *_Metric) Update(d *schema.ResourceData) error {
	return m.Save(d)
}

func (m *_Metric) SetTags(tags _Tags) {
	m.Tags = tagsToAPI(tags)
}
