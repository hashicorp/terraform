package circonus

// The `circonus_metric` type is a synthetic, top-level resource that doesn't
// actually exist within Circonus.  The `circonus_check` resource uses
// `circonus_metric` as input to its `metric` attribute.  The `circonus_check`
// resource can, if configured, override various parameters in the
// `circonus_metric` resource if no value was set (e.g. the `icmp_ping` will
// implicitly set the `unit` metric to `seconds`).

import (
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_metric.* resource attribute names
	metricActiveAttr = "active"
	metricIDAttr     = "id"
	metricNameAttr   = "name"
	metricTypeAttr   = "type"
	metricTagsAttr   = "tags"
	metricUnitAttr   = "unit"

	// CheckBundle.Metric.Status can be one of these values
	metricStatusActive    = "active"
	metricStatusAvailable = "available"
)

var metricDescriptions = attrDescrs{
	metricActiveAttr: "Enables or disables the metric",
	metricNameAttr:   "Name of the metric",
	metricTypeAttr:   "Type of metric (e.g. numeric, histogram, text)",
	metricTagsAttr:   "Tags assigned to the metric",
	metricUnitAttr:   "The unit of measurement for a metric",
}

func resourceMetric() *schema.Resource {
	return &schema.Resource{
		Create: metricCreate,
		Read:   metricRead,
		Update: metricUpdate,
		Delete: metricDelete,
		Exists: metricExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(metricDescriptions, map[schemaAttr]*schema.Schema{
			metricActiveAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			metricNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(metricNameAttr, `[\S]+`),
			},
			metricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateStringIn(metricTypeAttr, validMetricTypes),
			},
			metricTagsAttr: tagMakeConfigSchema(metricTagsAttr),
			metricUnitAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      metricUnit,
				ValidateFunc: validateRegexp(metricUnitAttr, metricUnitRegexp),
			},
		}),
	}
}

func metricCreate(d *schema.ResourceData, meta interface{}) error {
	m := newMetric()

	id := d.Id()
	if id == "" {
		var err error
		id, err = newMetricID()
		if err != nil {
			return errwrap.Wrapf("metric ID creation failed: {{err}}", err)
		}
	}

	if err := m.ParseConfig(id, d); err != nil {
		return errwrap.Wrapf("error parsing metric schema during create: {{err}}", err)
	}

	if err := m.Create(d); err != nil {
		return errwrap.Wrapf("error creating metric: {{err}}", err)
	}

	return metricRead(d, meta)
}

func metricRead(d *schema.ResourceData, meta interface{}) error {
	m := newMetric()

	if err := m.ParseConfig(d.Id(), d); err != nil {
		return errwrap.Wrapf("error parsing metric schema during read: {{err}}", err)
	}

	if err := m.SaveState(d); err != nil {
		return errwrap.Wrapf("error saving metric during read: {{err}}", err)
	}

	return nil
}

func metricUpdate(d *schema.ResourceData, meta interface{}) error {
	m := newMetric()

	if err := m.ParseConfig(d.Id(), d); err != nil {
		return errwrap.Wrapf("error parsing metric schema during update: {{err}}", err)
	}

	if err := m.Update(d); err != nil {
		return errwrap.Wrapf("error updating metric: {{err}}", err)
	}

	return nil
}

func metricDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func metricExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if id := d.Id(); id != "" {
		return true, nil
	}

	return false, nil
}
