package ns1

import (
	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func apikeyResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"key": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"teams": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
	}
	s = addPermsSchema(s)
	return &schema.Resource{
		Schema: s,
		Create: ApikeyCreate,
		Read:   ApikeyRead,
		Update: ApikeyUpdate,
		Delete: ApikeyDelete,
	}
}

func apikeyToResourceData(d *schema.ResourceData, k *account.APIKey) error {
	d.SetId(k.ID)
	d.Set("name", k.Name)
	d.Set("key", k.Key)
	d.Set("teams", k.TeamIDs)
	permissionsToResourceData(d, k.Permissions)
	return nil
}

func resourceDataToApikey(k *account.APIKey, d *schema.ResourceData) error {
	k.ID = d.Id()
	k.Name = d.Get("name").(string)
	if v, ok := d.GetOk("teams"); ok {
		teamsRaw := v.([]interface{})
		k.TeamIDs = make([]string, len(teamsRaw))
		for i, team := range teamsRaw {
			k.TeamIDs[i] = team.(string)
		}
	} else {
		k.TeamIDs = make([]string, 0)
	}
	k.Permissions = resourceDataToPermissions(d)
	return nil
}

// ApikeyCreate creates ns1 API key
func ApikeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	k := account.APIKey{}
	if err := resourceDataToApikey(&k, d); err != nil {
		return err
	}
	if _, err := client.APIKeys.Create(&k); err != nil {
		return err
	}
	return apikeyToResourceData(d, &k)
}

// ApikeyRead reads API key from ns1
func ApikeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	k, _, err := client.APIKeys.Get(d.Id())
	if err != nil {
		return err
	}
	return apikeyToResourceData(d, k)
}

//ApikeyDelete deletes the given ns1 api key
func ApikeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	_, err := client.APIKeys.Delete(d.Id())
	d.SetId("")
	return err
}

//ApikeyUpdate updates the given api key in ns1
func ApikeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	k := account.APIKey{
		ID: d.Id(),
	}
	if err := resourceDataToApikey(&k, d); err != nil {
		return err
	}
	if _, err := client.APIKeys.Update(&k); err != nil {
		return err
	}
	return apikeyToResourceData(d, &k)
}
