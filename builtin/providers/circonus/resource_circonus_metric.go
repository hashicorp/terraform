package circonus

import (
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/schema"
)

type metric struct {
	ID   typeMetricID
	Name typeMetricName
	Tags typeTags
	Unit typeUnit
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

		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			metricNameAttr: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: metricDescription[metricNameAttr],
			},
			metricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateStringIn(metricTypeAttr, validMetricTypes),
				Description:  metricDescription[metricTypeAttr],
			},
			metricTagsAttr: &schema.Schema{
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateFunc:     validateTags,
				DiffSuppressFunc: suppressAutoTag,
				Description:      metricDescription[metricTagsAttr],
			},
			metricUnitAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: metricDescription[metricUnitAttr],
			},
		}),
	}
}

func metricCreate(d *schema.ResourceData, meta interface{}) error {
	m, err := metricSchemaParse(d, meta)
	if err != nil {
		return errwrap.Wrapf("error parsing metric schema during create: {{err}}", err)
	}

	if err = metricSchemaSave(d, m); err != nil {
		return errwrap.Wrapf("error saving metric during create: {{err}}", err)
	}

	return metricRead(d, meta)
}

func metricRead(d *schema.ResourceData, meta interface{}) error {
	m, err := metricSchemaParse(d, meta)
	if err != nil {
		return errwrap.Wrapf("unable to read metric during read: {{err}}", err)
	}

	if err = metricSchemaSave(d, m); err != nil {
		return errwrap.Wrapf("error saving metric during read: {{err}}", err)
	}

	return nil
}

func metricUpdate(d *schema.ResourceData, meta interface{}) error {
	m, err := metricSchemaParse(d, meta)
	if err != nil {
		return errwrap.Wrapf("unable to read metric during read: {{err}}", err)
	}

	if err = metricSchemaSave(d, m); err != nil {
		return errwrap.Wrapf("error saving metric during read: {{err}}", err)
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

func metricSchemaParse(d *schema.ResourceData, meta interface{}) (*metric, error) {
	ctxt := meta.(*providerContext)

	id := typeMetricID(schemaGetString(d, metricIDAttr))
	if id == "" {
		newID, err := uuid.GenerateUUID()
		if err != nil {
			return nil, errwrap.Wrapf("metric ID creation failed: {{err}}", err)
		}

		id = typeMetricID(newID)
	}

	name := typeMetricName(schemaGetString(d, metricNameAttr))
	unit := typeUnit(schemaGetString(d, metricUnitAttr))
	tags := schemaGetTags(ctxt, d, metricTagsAttr, typeTag{})

	return &metric{
		ID:   id,
		Name: name,
		Unit: unit,
		Tags: tags,
	}, nil
}

func metricSchemaSave(d *schema.ResourceData, m *metric) error {
	stateSet(d, metricNameAttr, m.Name)
	stateSet(d, metricTagsAttr, m.Tags)
	stateSet(d, metricUnitAttr, m.Unit)

	d.SetId(string(m.ID))

	return nil
}
