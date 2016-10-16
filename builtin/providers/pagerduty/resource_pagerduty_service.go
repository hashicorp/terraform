package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyService() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyServiceCreate,
		Read:   resourcePagerDutyServiceRead,
		Update: resourcePagerDutyServiceUpdate,
		Delete: resourcePagerDutyServiceDelete,
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyServiceImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"auto_resolve_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"acknowledgement_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"escalation_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func buildServiceStruct(d *schema.ResourceData) *pagerduty.Service {
	service := pagerduty.Service{
		Name: d.Get("name").(string),
	}

	if attr, ok := d.GetOk("description"); ok {
		service.Description = attr.(string)
	}

	if attr, ok := d.GetOk("auto_resolve_timeout"); ok {
		autoResolveTimeout := uint(attr.(int))
		service.AutoResolveTimeout = &autoResolveTimeout
	}

	if attr, ok := d.GetOk("acknowledgement_timeout"); ok {
		acknowledgementTimeout := uint(attr.(int))
		service.AcknowledgementTimeout = &acknowledgementTimeout
	}

	policy := &pagerduty.EscalationPolicy{}
	policy.ID = d.Get("escalation_policy").(string)
	policy.Type = "escalation_policy"
	service.EscalationPolicy = *policy

	return &service
}

func resourcePagerDutyServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	s := buildServiceStruct(d)

	log.Printf("[INFO] Creating PagerDuty service %s", s.Name)

	s, err := client.CreateService(*s)

	if err != nil {
		return err
	}

	d.SetId(s.ID)

	return nil
}

func resourcePagerDutyServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty service %s", d.Id())

	s, err := client.GetService(d.Id(), pagerduty.GetServiceOptions{})

	if err != nil {
		return err
	}

	d.Set("name", s.Name)
	d.Set("escalation_policy", s.EscalationPolicy.ID)
	d.Set("description", s.Description)
	d.Set("auto_resolve_timeout", s.AutoResolveTimeout)
	d.Set("acknowledgement_timeout", s.AcknowledgementTimeout)

	return nil
}

func resourcePagerDutyServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	s := buildServiceStruct(d)
	s.ID = d.Id()

	log.Printf("[INFO] Updating PagerDuty service %s", d.Id())

	s, err := client.UpdateService(*s)

	if err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty service %s", d.Id())

	err := client.DeleteService(d.Id())

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyServiceImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourcePagerDutyServiceRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
