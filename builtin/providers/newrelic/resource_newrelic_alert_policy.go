package newrelic

import (
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func resourceNewRelicAlertPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceNewRelicAlertPolicyCreate,
		Read:   resourceNewRelicAlertPolicyRead,
		// Update: Not currently supported in API
		Delete: resourceNewRelicAlertPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"incident_preference": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "PER_POLICY",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"PER_POLICY", "PER_CONDITION", "PER_CONDITION_AND_TARGET"}, false),
			},
			"created_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func buildAlertPolicyStruct(d *schema.ResourceData) *newrelic.AlertPolicy {
	policy := newrelic.AlertPolicy{
		Name: d.Get("name").(string),
	}

	if attr, ok := d.GetOk("incident_preference"); ok {
		policy.IncidentPreference = attr.(string)
	}

	return &policy
}

func resourceNewRelicAlertPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)
	policy := buildAlertPolicyStruct(d)

	log.Printf("[INFO] Creating New Relic alert policy %s", policy.Name)

	policy, err := client.CreateAlertPolicy(*policy)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(policy.ID))

	return nil
}

func resourceNewRelicAlertPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading New Relic alert policy %v", id)

	policy, err := client.GetAlertPolicy(int(id))
	if err != nil {
		if err == newrelic.ErrNotFound {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", policy.Name)
	d.Set("incident_preference", policy.IncidentPreference)
	d.Set("created_at", policy.CreatedAt)
	d.Set("updated_at", policy.UpdatedAt)

	return nil
}

func resourceNewRelicAlertPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting New Relic alert policy %v", id)

	if err := client.DeleteAlertPolicy(int(id)); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
