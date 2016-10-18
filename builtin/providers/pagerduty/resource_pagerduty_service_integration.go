package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyServiceIntegration() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyServiceIntegrationCreate,
		Read:   resourcePagerDutyServiceIntegrationRead,
		Update: resourcePagerDutyServiceIntegrationUpdate,

		// NOTE: It's currently not possible to delete integrations via the API.
		// Therefore it needs to be manually removed from the Web UI.
		Delete: resourcePagerDutyServiceIntegrationDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validateValueFunc([]string{
					"aws_cloudwatch_inbound_integration",
					"cloudkick_inbound_integration",
					"event_transformer_api_inbound_integration",
					"generic_email_inbound_integration",
					"generic_events_api_inbound_integration",
					"keynote_inbound_integration",
					"nagios_inbound_integration",
					"pingdom_inbound_integration",
					"sql_monitor_inbound_integration",
				}),
			},
			"integration_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"integration_email": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func buildServiceIntegrationStruct(d *schema.ResourceData) *pagerduty.Integration {
	service := pagerduty.Integration{
		Type: d.Get("type").(string),
		Name: d.Get("name").(string),
		Service: &pagerduty.APIObject{
			Type: "service",
			ID:   d.Get("service").(string),
		},
		APIObject: pagerduty.APIObject{
			ID: d.Id(),
		},
	}

	return &service
}

func resourcePagerDutyServiceIntegrationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	i := buildServiceIntegrationStruct(d)

	log.Printf("[INFO] Creating PagerDuty service integration %s", i.Name)

	service := d.Get("service").(string)

	s, err := client.CreateIntegration(service, *i)

	if err != nil {
		return err
	}

	d.SetId(s.ID)

	return resourcePagerDutyServiceIntegrationRead(d, meta)
}

func resourcePagerDutyServiceIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty service integration %s", d.Id())

	service := d.Get("service").(string)

	i, err := client.GetIntegration(service, d.Id(), pagerduty.GetIntegrationOptions{})

	if err != nil {
		return err
	}

	d.Set("name", i.Name)
	d.Set("type", i.Type)
	d.Set("service", i.Service)
	d.Set("integration_key", i.IntegrationKey)
	d.Set("integration_email", i.IntegrationEmail)

	return nil
}

func resourcePagerDutyServiceIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	s := buildServiceIntegrationStruct(d)

	service := d.Get("service").(string)

	log.Printf("[INFO] Updating PagerDuty service integration %s", d.Id())

	s, err := client.UpdateIntegration(service, *s)

	if err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyServiceIntegrationDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Removing PagerDuty service integration %s", d.Id())

	d.SetId("")

	return nil
}
