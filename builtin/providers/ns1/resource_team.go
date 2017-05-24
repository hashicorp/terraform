package ns1

import (
	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func teamResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
	}
	s = addPermsSchema(s)
	return &schema.Resource{
		Schema: s,
		Create: TeamCreate,
		Read:   TeamRead,
		Update: TeamUpdate,
		Delete: TeamDelete,
	}
}

func teamToResourceData(d *schema.ResourceData, t *account.Team) error {
	d.SetId(t.ID)
	d.Set("name", t.Name)
	permissionsToResourceData(d, t.Permissions)
	return nil
}

func resourceDataToTeam(t *account.Team, d *schema.ResourceData) error {
	t.ID = d.Id()
	t.Name = d.Get("name").(string)
	t.Permissions = resourceDataToPermissions(d)
	return nil
}

// TeamCreate creates the given team in ns1
func TeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	t := account.Team{}
	if err := resourceDataToTeam(&t, d); err != nil {
		return err
	}
	if _, err := client.Teams.Create(&t); err != nil {
		return err
	}
	return teamToResourceData(d, &t)
}

// TeamRead reads the team data from ns1
func TeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	t, _, err := client.Teams.Get(d.Id())
	if err != nil {
		return err
	}
	return teamToResourceData(d, t)
}

// TeamDelete deletes the given team from ns1
func TeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	_, err := client.Teams.Delete(d.Id())
	d.SetId("")
	return err
}

// TeamUpdate updates the given team in ns1
func TeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	t := account.Team{
		ID: d.Id(),
	}
	if err := resourceDataToTeam(&t, d); err != nil {
		return err
	}
	if _, err := client.Teams.Update(&t); err != nil {
		return err
	}
	return teamToResourceData(d, &t)
}
