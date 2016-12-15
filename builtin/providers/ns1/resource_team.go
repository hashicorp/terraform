package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func teamResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1TeamCreate,
		Read:   resourceNS1TeamRead,
		Update: resourceNS1TeamUpdate,
		Delete: resourceNS1TeamDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"permissions": permissionsSchema(),
		},
	}
}

func resourceNS1TeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	t := buildNS1Team(d)

	log.Printf("[INFO] Creating NS1 team: %s \n", t.Name)

	if _, err := client.Teams.Create(t); err != nil {
		return err
	}

	d.SetId(t.ID)

	return resourceNS1TeamRead(d, meta)
}

func resourceNS1TeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 team: %s \n", d.Id())

	t, _, err := client.Teams.Get(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", t.Name)
	if err := d.Set("permissions", flattenNS1Permissions(t.Permissions)); err != nil {
		return err
	}

	return nil
}

func resourceNS1TeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	t := buildNS1Team(d)
	t.ID = d.Id()

	log.Printf("[INFO] Updating NS1 team: %s \n", t.ID)

	if _, err := client.Teams.Update(t); err != nil {
		return err
	}

	return nil
}

func resourceNS1TeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 team: %s \n", d.Id())

	if _, err := client.Teams.Delete(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1Team(d *schema.ResourceData) *account.Team {
	return &account.Team{
		Name:        d.Get("name").(string),
		Permissions: expandNS1Permissions(d),
	}
}
