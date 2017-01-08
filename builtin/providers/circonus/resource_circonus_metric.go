package circonus

// The `circonus_metric` type is a synthetic, top-level resource that doesn't
// actually exist within Circonus.  The `circonus_check` resource uses
// `circonus_metric` as input to its `streams` attribute.  The `circonus_check`
// resource can, if configured, override various parameters in the
// `circonus_metric` resource if no value was set (e.g. the `icmp_ping` will
// implicitly set the `unit` metric to `seconds`).

import (
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

// circonus_metric.* resource attribute names
const (
	_MetricIDAttr   _SchemaAttr = "id"
	_MetricNameAttr _SchemaAttr = "name"
	_MetricTypeAttr _SchemaAttr = "type"
	_MetricTagsAttr _SchemaAttr = "tags"
	_MetricUnitAttr _SchemaAttr = "unit"
)

var _MetricDescriptions = _AttrDescrs{
	_MetricNameAttr: "Name of the metric",
	_MetricTypeAttr: "Type of metric",
}

func _NewCirconusMetricResource() *schema.Resource {
	return &schema.Resource{
		Create: _MetricCreate,
		Read:   _MetricRead,
		Update: _MetricUpdate,
		Delete: _MetricDelete,
		Exists: _MetricExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: castSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_MetricNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			_MetricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateStringIn(_MetricTypeAttr, _ValidMetricTypes),
			},
			_MetricTagsAttr: &schema.Schema{
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateFunc:     validateTags,
				DiffSuppressFunc: suppressAutoTag,
			},
			_MetricUnitAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		}, _MetricDescriptions),
	}
}

func _MetricCreate(d *schema.ResourceData, meta interface{}) error {
	m := _NewMetric()
	if err := m.ParseSchema(d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during create: {{err}}", err)
	}

	if err := m.Save(d); err != nil {
		return errwrap.Wrapf("error saving metric during create: {{err}}", err)
	}

	return _MetricRead(d, meta)
}

func _MetricRead(d *schema.ResourceData, meta interface{}) error {
	m := _NewMetric()
	if err := m.ParseSchema(d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during read: {{err}}", err)
	}

	if err := m.Save(d); err != nil {
		return errwrap.Wrapf("error saving metric during read: {{err}}", err)
	}

	return nil
}

func _MetricUpdate(d *schema.ResourceData, meta interface{}) error {
	m := _NewMetric()
	if err := m.ParseSchema(d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during update: {{err}}", err)
	}

	if err := m.Save(d); err != nil {
		return errwrap.Wrapf("error saving metric during update: {{err}}", err)
	}

	return nil
}

func _MetricDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func _MetricExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if id := d.Id(); id != "" {
		return true, nil
	}

	return false, nil
}
