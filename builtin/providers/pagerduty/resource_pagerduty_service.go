package pagerduty

import (
	"log"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyService() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyServiceCreate,
		Read:   resourcePagerDutyServiceRead,
		Update: resourcePagerDutyServiceUpdate,
		Delete: resourcePagerDutyServiceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"auto_resolve_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"last_incident_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"acknowledgement_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"escalation_policy": {
				Type:     schema.TypeString,
				Required: true,
			},
			"incident_urgency_rule": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"urgency": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"during_support_hours": {
							Type:     schema.TypeList,
							MaxItems: 1,
							MinItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"urgency": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"outside_support_hours": {
							Type:     schema.TypeList,
							MaxItems: 1,
							MinItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"urgency": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"support_hours": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				MinItems: 1,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"time_zone": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"start_time": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"end_time": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"days_of_week": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 7,
							Elem:     &schema.Schema{Type: schema.TypeInt},
						},
					},
				},
			},
			"scheduled_actions": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"to_urgency": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"at": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildServiceStruct(d *schema.ResourceData) *pagerduty.Service {
	service := pagerduty.Service{
		Name:   d.Get("name").(string),
		Status: d.Get("status").(string),
		APIObject: pagerduty.APIObject{
			ID: d.Id(),
		},
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

	escalationPolicy := &pagerduty.EscalationPolicy{
		APIObject: pagerduty.APIObject{
			ID:   d.Get("escalation_policy").(string),
			Type: "escalation_policy_reference",
		},
	}

	service.EscalationPolicy = *escalationPolicy

	if attr, ok := d.GetOk("incident_urgency_rule"); ok {
		if iur, ok := expandIncidentUrgencyRule(attr); ok {
			service.IncidentUrgencyRule = iur
		}
	}
	if attr, ok := d.GetOk("support_hours"); ok {
		service.SupportHours = expandSupportHours(attr)
	}
	if attr, ok := d.GetOk("scheduled_actions"); ok {
		service.ScheduledActions = expandScheduledActions(attr)
	}

	return &service
}

func resourcePagerDutyServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	service := buildServiceStruct(d)

	log.Printf("[INFO] Creating PagerDuty service %s", service.Name)

	service, err := client.CreateService(*service)

	if err != nil {
		return err
	}

	d.SetId(service.ID)

	return nil
}

func resourcePagerDutyServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty service %s", d.Id())

	o := &pagerduty.GetServiceOptions{}

	service, err := client.GetService(d.Id(), o)

	if err != nil {
		if isNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", service.Name)
	d.Set("status", service.Status)
	d.Set("created_at", service.CreateAt)
	d.Set("escalation_policy", service.EscalationPolicy.ID)
	d.Set("description", service.Description)
	d.Set("auto_resolve_timeout", service.AutoResolveTimeout)
	d.Set("last_incident_timestamp", service.LastIncidentTimestamp)
	d.Set("acknowledgement_timeout", service.AcknowledgementTimeout)

	if incidentUrgencyRule, ok := flattenIncidentUrgencyRule(service); ok {
		d.Set("incident_urgency_rule", incidentUrgencyRule)
	}

	supportHours := flattenSupportHours(service)
	d.Set("support_hours", supportHours)

	scheduledActions := flattenScheduledActions(service)
	d.Set("scheduled_actions", scheduledActions)

	return nil
}

func resourcePagerDutyServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	service := buildServiceStruct(d)

	log.Printf("[INFO] Updating PagerDuty service %s", d.Id())

	if _, err := client.UpdateService(*service); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty service %s", d.Id())

	if err := client.DeleteService(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
