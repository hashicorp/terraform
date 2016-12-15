package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func apikeyResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1ApikeyCreate,
		Read:   resourceNS1ApikeyRead,
		Update: resourceNS1ApikeyUpdate,
		Delete: resourceNS1ApikeyDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"key": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"teams": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"permissions": permissionsSchema(),
		},
	}
}

func resourceNS1ApikeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	k := buildNS1Apikey(d)

	log.Printf("[INFO] Creating NS1 apikey: %#v \n", k)

	if _, err := client.APIKeys.Create(k); err != nil {
		return err
	}

	d.SetId(k.ID)

	return resourceNS1ApikeyRead(d, meta)
}

func resourceNS1ApikeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 apikey: %s \n", d.Id())

	k, _, err := client.APIKeys.Get(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", k.Name)
	d.Set("key", k.Key)
	d.Set("teams", k.TeamIDs)
	if err := d.Set("permissions", flattenNS1Permissions(k.Permissions)); err != nil {
		return err
	}

	return nil
}

func resourceNS1ApikeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	k := buildNS1Apikey(d)
	k.ID = d.Id()

	log.Printf("[INFO] Updating NS1 apikey: %s \n", k.ID)

	if _, err := client.APIKeys.Update(k); err != nil {
		return err
	}

	return nil
}

func resourceNS1ApikeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 apikey: %s \n", d.Id())

	if _, err := client.APIKeys.Delete(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1Apikey(d *schema.ResourceData) *account.APIKey {
	k := &account.APIKey{
		Name:        d.Get("name").(string),
		Permissions: expandNS1Permissions(d),
	}

	teams := d.Get("teams").([]interface{})
	k.TeamIDs = make([]string, len(teams))
	for i, t := range teams {
		k.TeamIDs[i] = t.(string)
	}

	return k
}
