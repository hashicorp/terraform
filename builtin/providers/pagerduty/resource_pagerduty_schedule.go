package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
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
				Type:     schema.TypeList,
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
							Computed: true,
						},
						"start": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if old == "" {
									return false
								}
								return true
							},
						},
						"end": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"rotation_virtual_start": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if old == "" {
									return false
								}
								return true
							},
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
							Optional: true,
							Type:     schema.TypeList,
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
	scheduleLayers := d.Get("schedule_layer").([]interface{})

	schedule := pagerduty.Schedule{
		Name:           d.Get("name").(string),
		TimeZone:       d.Get("time_zone").(string),
		ScheduleLayers: expandLayers(scheduleLayers),
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

	return resourcePagerDutyScheduleRead(d, meta)
}

func resourcePagerDutyScheduleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty schedule: %s", d.Id())

	s, err := client.GetSchedule(d.Id(), pagerduty.GetScheduleOptions{})

	if err != nil {
		return err
	}

	d.Set("name", s.Name)
	d.Set("time_zone", s.TimeZone)
	d.Set("description", s.Description)

	if err := d.Set("schedule_layer", flattenLayers(s.ScheduleLayers)); err != nil {
		return err
	}

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
