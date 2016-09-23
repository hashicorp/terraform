package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyTeamCreate,
		Read:   resourcePagerDutyTeamRead,
		Update: resourcePagerDutyTeamUpdate,
		Delete: resourcePagerDutyTeamDelete,
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyTeamImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
		},
	}
}

func buildTeamStruct(d *schema.ResourceData) *pagerduty.Team {
	team := pagerduty.Team{
		Name: d.Get("name").(string),
	}

	if attr, ok := d.GetOk("description"); ok {
		team.Description = attr.(string)
	}

	return &team
}

func resourcePagerDutyTeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	t := buildTeamStruct(d)

	log.Printf("[INFO] Creating PagerDuty team %s", t.Name)

	t, err := client.CreateTeam(t)

	if err != nil {
		return err
	}

	d.SetId(t.ID)

	return nil

}

func resourcePagerDutyTeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty team %s", d.Id())

	t, err := client.GetTeam(d.Id())

	if err != nil {
		return err
	}

	d.Set("name", t.Name)
	d.Set("description", t.Description)

	return nil
}

func resourcePagerDutyTeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	t := buildTeamStruct(d)

	log.Printf("[INFO] Updating PagerDuty team %s", d.Id())

	t, err := client.UpdateTeam(d.Id(), t)

	if err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyTeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty team %s", d.Id())

	err := client.DeleteTeam(d.Id())

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyTeamImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourcePagerDutyTeamRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
