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
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"time_zone": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"layer": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"start": {
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
						"end": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"rotation_virtual_start": {
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
						"rotation_turn_length_seconds": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"users": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"restriction": {
							Optional: true,
							Type:     schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"start_time_of_day": {
										Type:     schema.TypeString,
										Required: true,
									},
									"start_day_of_week": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"duration_seconds": {
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

func buildScheduleStruct(d *schema.ResourceData) *pagerduty.Schedule {
	scheduleLayers := d.Get("layer").([]interface{})

	schedule := pagerduty.Schedule{
		Name:           d.Get("name").(string),
		TimeZone:       d.Get("time_zone").(string),
		ScheduleLayers: expandScheduleLayers(scheduleLayers),
	}

	if attr, ok := d.GetOk("description"); ok {
		schedule.Description = attr.(string)
	}

	return &schedule
}

func resourcePagerDutyScheduleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	schedule := buildScheduleStruct(d)

	log.Printf("[INFO] Creating PagerDuty schedule: %s", schedule.Name)

	schedule, err := client.CreateSchedule(*schedule)

	if err != nil {
		return err
	}

	d.SetId(schedule.ID)

	return resourcePagerDutyScheduleRead(d, meta)
}

func resourcePagerDutyScheduleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty schedule: %s", d.Id())

	schedule, err := client.GetSchedule(d.Id(), pagerduty.GetScheduleOptions{})

	if err != nil {
		return err
	}

	d.Set("name", schedule.Name)
	d.Set("time_zone", schedule.TimeZone)
	d.Set("description", schedule.Description)

	if err := d.Set("layer", flattenScheduleLayers(schedule.ScheduleLayers)); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyScheduleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	schedule := buildScheduleStruct(d)

	log.Printf("[INFO] Updating PagerDuty schedule: %s", d.Id())

	if _, err := client.UpdateSchedule(d.Id(), *schedule); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyScheduleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty schedule: %s", d.Id())

	if err := client.DeleteSchedule(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
