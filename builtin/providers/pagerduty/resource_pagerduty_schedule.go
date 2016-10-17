package pagerduty

import (
	"bytes"
	"fmt"
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutySchedule() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyScheduleCreate,
		Read:   resourcePagerDutyScheduleRead,
		Update: resourcePagerDutyScheduleUpdate,
		Delete: resourcePagerDutyScheduleDelete,
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyScheduleImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"schedule_layer": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      resourcePagerDutyEscalationHash,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"start": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"end": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"rotation_virtual_start": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"rotation_turn_length_seconds": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"users": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"restriction": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"start_time_of_day": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"duration_seconds": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
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

func buildScheduleStruct(d *schema.ResourceData) (*pagerduty.Schedule, error) {
	pagerdutyLayers := d.Get("schedule_layer").(*schema.Set).List()

	schedule := pagerduty.Schedule{
		Name:           d.Get("name").(string),
		TimeZone:       d.Get("time_zone").(string),
		ScheduleLayers: expandLayers(pagerdutyLayers),
	}

	if attr, ok := d.GetOk("description"); ok {
		schedule.Description = attr.(string)
	}

	return &schedule, nil
}

func resourcePagerDutyScheduleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	s, _ := buildScheduleStruct(d)

	log.Printf("[INFO] Creating PagerDuty schedule: %s", s.Name)

	e, err := client.CreateSchedule(*s)

	if err != nil {
		return err
	}

	d.SetId(e.ID)

	return nil
}

func resourcePagerDutyScheduleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty schedule: %s", d.Id())

	s, err := client.GetSchedule(d.Id(), pagerduty.GetScheduleOptions{})

	if err != nil {
		return err
	}

	d.Set("name", s.Name)
	d.Set("description", s.Description)
	d.Set("schedule_layer", flattenLayers(s.ScheduleLayers))

	return nil
}

func resourcePagerDutyScheduleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	e, _ := buildScheduleStruct(d)

	log.Printf("[INFO] Updating PagerDuty schedule: %s", d.Id())

	e, err := client.UpdateSchedule(d.Id(), *e)

	if err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyScheduleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty schedule: %s", d.Id())

	err := client.DeleteSchedule(d.Id())

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyScheduleImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourcePagerDutyScheduleRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourcePagerDutyEscalationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["rotation_turn_length_seconds"].(int)))

	if _, ok := m["name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	}

	if _, ok := m["end"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", m["end"].(string)))
	}

	for _, u := range m["users"].([]interface{}) {
		buf.WriteString(fmt.Sprintf("%s-", u))
	}

	for _, r := range m["restriction"].([]interface{}) {
		restriction := r.(map[string]interface{})
		buf.WriteString(fmt.Sprintf("%s-", restriction["type"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", restriction["start_time_of_day"].(string)))
		buf.WriteString(fmt.Sprintf("%d-", restriction["duration_seconds"].(int)))
	}

	return hashcode.String(buf.String())
}
