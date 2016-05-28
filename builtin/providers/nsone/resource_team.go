package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
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

func teamToResourceData(d *schema.ResourceData, t *nsone.Team) error {
	d.SetId(t.Id)
	d.Set("name", t.Name)
	permissionsToResourceData(d, t.Permissions)
	return nil
}

func resourceDataToTeam(u *nsone.Team, d *schema.ResourceData) error {
	u.Id = d.Id()
	u.Name = d.Get("name").(string)
	u.Permissions = resourceDataToPermissions(d)
	return nil
}

// TeamCreate creates the given team in ns1
func TeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.Team{}
	if err := resourceDataToTeam(&mj, d); err != nil {
		return err
	}
	if err := client.CreateTeam(&mj); err != nil {
		return err
	}
	return teamToResourceData(d, &mj)
}

// TeamRead reads the team data from ns1
func TeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetTeam(d.Id())
	if err != nil {
		return err
	}
	teamToResourceData(d, &mj)
	return nil
}

// TeamDelete deletes the given team from ns1
func TeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteTeam(d.Id())
	d.SetId("")
	return err
}

// TeamUpdate updates the given team in ns1
func TeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.Team{
		Id: d.Id(),
	}
	if err := resourceDataToTeam(&mj, d); err != nil {
		return err
	}
	if err := client.UpdateTeam(&mj); err != nil {
		return err
	}
	teamToResourceData(d, &mj)
	return nil
}
