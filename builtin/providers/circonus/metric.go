package circonus

// The _Metric type is the backing store of the `circonus_metric` resource.

import (
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/schema"
)

type _Metric struct {
	ID   _MetricID
	Name _MetricName
	Tags _Tags
	Unit _Unit
}

func _NewMetric() _Metric {
	return _Metric{}
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

	m.ID = _MetricID(id)
	m.Name = _MetricName(schemaGetString(d, _MetricNameAttr))
	m.Unit = _Unit(schemaGetString(d, _MetricUnitAttr))
	m.Tags = schemaGetTags(ctxt, d, _MetricTagsAttr, _Tag{})

	return nil
}

func (m *_Metric) Save(d *schema.ResourceData) error {
	stateSet(d, _MetricNameAttr, m.Name)
	stateSet(d, _MetricTagsAttr, m.Tags)
	stateSet(d, _MetricUnitAttr, m.Unit)

	d.SetId(string(m.ID))

	return nil
}
