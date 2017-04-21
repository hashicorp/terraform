package pagerduty

import (
	"fmt"
	"log"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutySchedule() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyScheduleRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourcePagerDutyScheduleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty schedule")

	searchName := d.Get("name").(string)

	o := &pagerduty.ListSchedulesOptions{
		Query: searchName,
	}

	resp, err := client.ListSchedules(*o)
	if err != nil {
		return err
	}

	var found *pagerduty.Schedule

	for _, schedule := range resp.Schedules {
		if schedule.Name == searchName {
			found = &schedule
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any schedule with the name: %s", searchName)
	}

	d.SetId(found.ID)
	d.Set("name", found.Name)

	return nil
}
