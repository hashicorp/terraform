package pagerduty

import (
	"fmt"
	"log"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyServiceIntegration() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyServiceIntegrationCreate,
		Read:   resourcePagerDutyServiceIntegrationRead,
		Update: resourcePagerDutyServiceIntegrationUpdate,
		Delete: resourcePagerDutyServiceIntegrationDelete,
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyServiceIntegrationImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"service": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"vendor"},
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
			"vendor": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"type"},
				Computed:      true,
			},
			"integration_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"integration_email": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func buildServiceIntegrationStruct(d *schema.ResourceData) *pagerduty.Integration {
	serviceIntegration := pagerduty.Integration{
		Name: d.Get("name").(string),
		Service: &pagerduty.APIObject{
			Type: "service",
			ID:   d.Get("service").(string),
		},
		APIObject: pagerduty.APIObject{
			ID:   d.Id(),
			Type: "service_integration",
		},
	}

	if attr, ok := d.GetOk("integration_key"); ok {
		serviceIntegration.IntegrationKey = attr.(string)
	}

	if attr, ok := d.GetOk("integration_email"); ok {
		serviceIntegration.IntegrationEmail = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		serviceIntegration.Type = attr.(string)
	}

	if attr, ok := d.GetOk("vendor"); ok {
		serviceIntegration.Vendor = &pagerduty.APIObject{
			ID:   attr.(string),
			Type: "vendor",
		}
	}

	return &serviceIntegration
}

func resourcePagerDutyServiceIntegrationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	serviceIntegration := buildServiceIntegrationStruct(d)

	log.Printf("[INFO] Creating PagerDuty service integration %s", serviceIntegration.Name)

	service := d.Get("service").(string)

	serviceIntegration, err := client.CreateIntegration(service, *serviceIntegration)

	if err != nil {
		return err
	}

	d.SetId(serviceIntegration.ID)

	return resourcePagerDutyServiceIntegrationRead(d, meta)
}

func resourcePagerDutyServiceIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty service integration %s", d.Id())

	service := d.Get("service").(string)

	o := &pagerduty.GetIntegrationOptions{}

	serviceIntegration, err := client.GetIntegration(service, d.Id(), *o)

	if err != nil {
		if isNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", serviceIntegration.Name)
	d.Set("type", serviceIntegration.Type)
	d.Set("service", serviceIntegration.Service.ID)
	d.Set("vendor", serviceIntegration.Vendor.ID)
	d.Set("integration_key", serviceIntegration.IntegrationKey)
	d.Set("integration_email", serviceIntegration.IntegrationEmail)

	return nil
}

func resourcePagerDutyServiceIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	serviceIntegration := buildServiceIntegrationStruct(d)

	service := d.Get("service").(string)

	log.Printf("[INFO] Updating PagerDuty service integration %s", d.Id())

	if _, err := client.UpdateIntegration(service, *serviceIntegration); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyServiceIntegrationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	service := d.Get("service").(string)

	log.Printf("[INFO] Removing PagerDuty service integration %s", d.Id())

	if err := client.DeleteIntegration(service, d.Id()); err != nil {
		if isNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyServiceIntegrationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*pagerduty.Client)

	resp, err := client.ListServices(pagerduty.ListServiceOptions{})
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	var serviceID string

	for _, service := range resp.Services {
		for _, integration := range service.Integrations {
			if integration.ID == d.Id() {
				serviceID = service.ID
			}
		}
	}

	if serviceID == "" {
		return []*schema.ResourceData{}, fmt.Errorf("Error importing pagerduty_service_integration. Could not locate a service ID for the integration")
	}

	d.Set("service", serviceID)

	return []*schema.ResourceData{d}, nil
}
