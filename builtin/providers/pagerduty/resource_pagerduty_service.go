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
			State: resourcePagerDutyServiceImport,
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
			"summary": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"incident_urgency_rule": {
				Type:     schema.TypeMap,
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
							Type:     schema.TypeMap,
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
							Type:     schema.TypeMap,
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
			"support_hours": {
				Type:     schema.TypeMap,
				Optional: true,
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
						"days_of_week": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeInt},
						},
						"start_time": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"end_time": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"scheduled_actions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"named_time": {
							Type:     schema.TypeMap,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"to_urgency": {
							Type:     schema.TypeString,
							Required: true,
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

	iurMap := d.Get("incident_urgency_rule").(map[string]interface{})
	iur := pagerduty.IncidentUrgencyRule{}
	if val, ok := iurMap["during_support_hours"]; ok {
		var duringSupportHours pagerduty.IncidentUrgencyType
		m := val.(map[string]interface{})
		if val, ok := m["type"]; ok {
			duringSupportHours.Type = val.(string)
		}
		if val, ok := m["urgency"]; ok {
			duringSupportHours.Urgency = val.(string)
		}
		iur.DuringSupportHours = &duringSupportHours
	}
	if val, ok := iurMap["outside_support_hours"]; ok {
		var outsideSupportHours pagerduty.IncidentUrgencyType
		m := val.(map[string]interface{})
		if val, ok := m["type"]; ok {
			outsideSupportHours.Type = val.(string)
		}
		if val, ok := m["urgency"]; ok {
			outsideSupportHours.Urgency = val.(string)
		}
		iur.OutsideSupportHours = &outsideSupportHours
	}
	if val, ok := iurMap["type"]; ok {
		iur.Type = val.(string)
	}
	if val, ok := iurMap["urgency"]; ok {
		iur.Urgency = val.(string)
	}
	if iur.DuringSupportHours != nil ||
		iur.OutsideSupportHours != nil ||
		iur.Type != "" ||
		iur.Urgency != "" {
		service.IncidentUrgencyRule = &iur
	}

	if val, ok := d.GetOk("support_hours"); ok {
		var supportHours pagerduty.SupportHours
		m := val.(map[string]interface{})
		if val, ok := m["type"]; ok {
			supportHours.Type = val.(string)
		}
		if val, ok := m["time_zone"]; ok {
			supportHours.Timezone = val.(string)
		}
		if val, ok := m["start_time"]; ok {
			supportHours.StartTime = val.(string)
		}
		if val, ok := m["end_time"]; ok {
			supportHours.EndTime = val.(string)
		}
		supportHours.DaysOfWeek = getUintArrayFromMap(m, "days_of_week")
		if supportHours.Type != "" ||
			supportHours.Timezone != "" ||
			supportHours.StartTime != "" ||
			supportHours.EndTime != "" ||
			len(supportHours.DaysOfWeek) > 0 {
			service.SupportHours = &supportHours
		}
	}

	sa := d.Get("scheduled_actions")
	if sa != nil {
		var scheduledActions []pagerduty.ScheduledAction
		for _, e := range sa.(*schema.Set).List() {
			m := e.(map[string]interface{})
			var actionType string
			if val, ok := m["type"]; ok {
				actionType = val.(string)
			}
			var at pagerduty.InlineModel
			if val, ok := m["named_time"]; ok {
				n := val.(map[string]interface{})
				if val, ok := n["type"]; ok {
					at.Type = val.(string)
				}
				if val, ok := n["name"]; ok {
					at.Name = val.(string)
				}
			}
			var toUrgency string
			if val, ok := m["to_urgency"]; ok {
				toUrgency = val.(string)
			}
			scheduledAction := pagerduty.ScheduledAction{
				Type:      actionType,
				At:        at,
				ToUrgency: toUrgency,
			}
			scheduledActions = append(scheduledActions, scheduledAction)
		}
		service.ScheduledActions = scheduledActions
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
		return err
	}

	var duringSupportHours map[string]interface{}
	if service.IncidentUrgencyRule.DuringSupportHours != nil {
		duringSupportHours = map[string]interface{}{
			"type":    service.IncidentUrgencyRule.DuringSupportHours.Type,
			"urgency": service.IncidentUrgencyRule.DuringSupportHours.Urgency,
		}
	}
	var outsideSupportHours map[string]interface{}
	if service.IncidentUrgencyRule.OutsideSupportHours != nil {
		outsideSupportHours = map[string]interface{}{
			"type":    service.IncidentUrgencyRule.OutsideSupportHours.Type,
			"urgency": service.IncidentUrgencyRule.OutsideSupportHours.Urgency,
		}
	}
	incidentUrgencyRule := map[string]interface{}{
		"type":                  service.IncidentUrgencyRule.Type,
		"urgency":               service.IncidentUrgencyRule.Urgency,
		"during_support_hours":  duringSupportHours,
		"outside_support_hours": outsideSupportHours,
	}

	d.Set("name", service.Name)
	d.Set("status", service.Status)
	d.Set("created_at", service.CreateAt)
	d.Set("escalation_policy", service.EscalationPolicy.ID)
	d.Set("description", service.Description)
	d.Set("auto_resolve_timeout", service.AutoResolveTimeout)
	d.Set("last_incident_timestamp", service.LastIncidentTimestamp)
	d.Set("acknowledgement_timeout", service.AcknowledgementTimeout)
	d.Set("incident_urgency_rule", incidentUrgencyRule)
	d.Set("support_hours", service.SupportHours)
	d.Set("scheduled_actions", service.ScheduledActions)

	return nil
}

func resourcePagerDutyServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("name") ||
		d.HasChange("description") ||
		d.HasChange("auto_resolve_timeout") ||
		d.HasChange("escalation_policy_id") ||
		d.HasChange("incident_urgency_rule") ||
		d.HasChange("support_hours") ||
		d.HasChange("scheduled_actions") {

		client := meta.(*pagerduty.Client)
		service := buildServiceStruct(d)

		log.Printf("[INFO] Updating PagerDuty service %s", d.Id())

		if _, err := client.UpdateService(*service); err != nil {
			return err
		}
	}

	return resourcePagerDutyServiceRead(d, meta)
}

func resourcePagerDutyServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty service %s", d.Id())

	if err := client.DeleteService(d.Id()); err != nil {
		if "HTTP Status Code: 404" == err.Error() {
			d.SetId("")
			return nil
		}
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
