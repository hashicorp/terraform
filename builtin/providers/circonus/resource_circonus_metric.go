package circonus

// The `circonus_metric` type is a synthetic, top-level resource that doesn't
// actually exist within Circonus.  The `circonus_check` resource uses
// `circonus_metric` as input to its `streams` attribute.  The `circonus_check`
// resource can, if configured, override various parameters in the
// `circonus_metric` resource if no value was set (e.g. the `icmp_ping` will
// implicitly set the `unit` metric to `seconds`).

import (
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/schema"
)

// circonus_metric.* resource attribute names
const (
	_MetricActiveAttr _SchemaAttr = "active"
	_MetricIDAttr     _SchemaAttr = "id"
	_MetricNameAttr   _SchemaAttr = "name"
	_MetricTypeAttr   _SchemaAttr = "type"
	_MetricTagsAttr   _SchemaAttr = "tags"
	_MetricUnitAttr   _SchemaAttr = "unit"
)

const (
	// CheckBundle.Metric.Status can be one of these values
	_MetricStatusActive    = "active"
	_MetricStatusAvailable = "available"
)

var _MetricDescriptions = _AttrDescrs{
	_MetricActiveAttr: "Enables or disables the metric",
	_MetricNameAttr:   "Name of the metric",
	_MetricTypeAttr:   "Type of metric (e.g. numeric, histogram, text)",
	_MetricTagsAttr:   "Tags assigned to the metric",
	_MetricUnitAttr:   "The unit of measurement for a metric",
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
			_MetricActiveAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			_MetricNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(_MetricNameAttr, `[\S]+`),
			},
			_MetricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateStringIn(_MetricTypeAttr, _ValidMetricTypes),
			},
			_MetricTagsAttr: _TagMakeConfigSchema(_MetricTagsAttr),
			_MetricUnitAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(_MetricUnitAttr, `.+`),
			},
		}, _MetricDescriptions),
	}
}

func _MetricCreate(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	if id == "" {
		var err error
		id, err = uuid.GenerateUUID()
		if err != nil {
			return errwrap.Wrapf("metric ID creation failed: {{err}}", err)
		}
	}

	m := _NewMetric()
	if err := m.ParseConfig(id, d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during create: {{err}}", err)
	}

	if err := m.Create(d); err != nil {
		return errwrap.Wrapf("error creating metric: {{err}}", err)
	}

	return _MetricRead(d, meta)
}

func _MetricRead(d *schema.ResourceData, meta interface{}) error {
	m := _NewMetric()
	if err := m.ParseConfig(d.Id(), d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during read: {{err}}", err)
	}

	if err := m.SaveState(d); err != nil {
		return errwrap.Wrapf("error saving metric during read: {{err}}", err)
	}

	return nil
}

func _MetricUpdate(d *schema.ResourceData, meta interface{}) error {
	m := _NewMetric()

	if err := m.ParseConfig(d.Id(), d, meta); err != nil {
		return errwrap.Wrapf("error parsing metric schema during update: {{err}}", err)
	}

	if err := m.Update(d); err != nil {
		return errwrap.Wrapf("error updating metric: {{err}}", err)
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
