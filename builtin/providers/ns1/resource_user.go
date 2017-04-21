package ns1

import (
	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func userResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"username": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"email": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"notify": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Elem:     schema.TypeBool,
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
		Create: UserCreate,
		Read:   UserRead,
		Update: UserUpdate,
		Delete: UserDelete,
	}
}

func userToResourceData(d *schema.ResourceData, u *account.User) error {
	d.SetId(u.Username)
	d.Set("name", u.Name)
	d.Set("email", u.Email)
	d.Set("teams", u.TeamIDs)
	notify := make(map[string]bool)
	notify["billing"] = u.Notify.Billing
	d.Set("notify", notify)
	permissionsToResourceData(d, u.Permissions)
	return nil
}

func resourceDataToUser(u *account.User, d *schema.ResourceData) error {
	u.Name = d.Get("name").(string)
	u.Username = d.Get("username").(string)
	u.Email = d.Get("email").(string)
	if v, ok := d.GetOk("teams"); ok {
		teamsRaw := v.([]interface{})
		u.TeamIDs = make([]string, len(teamsRaw))
		for i, team := range teamsRaw {
			u.TeamIDs[i] = team.(string)
		}
	} else {
		u.TeamIDs = make([]string, 0)
	}
	if v, ok := d.GetOk("notify"); ok {
		notifyRaw := v.(map[string]interface{})
		u.Notify.Billing = notifyRaw["billing"].(bool)
	}
	u.Permissions = resourceDataToPermissions(d)
	return nil
}

// UserCreate creates the given user in ns1
func UserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	u := account.User{}
	if err := resourceDataToUser(&u, d); err != nil {
		return err
	}
	if _, err := client.Users.Create(&u); err != nil {
		return err
	}
	return userToResourceData(d, &u)
}

// UserRead  reads the given users data from ns1
func UserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	u, _, err := client.Users.Get(d.Id())
	if err != nil {
		return err
	}
	return userToResourceData(d, u)
}

// UserDelete deletes the given user from ns1
func UserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	_, err := client.Users.Delete(d.Id())
	d.SetId("")
	return err
}

// UserUpdate updates the user with given parameters in ns1
func UserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	u := account.User{
		Username: d.Id(),
	}
	if err := resourceDataToUser(&u, d); err != nil {
		return err
	}
	if _, err := client.Users.Update(&u); err != nil {
		return err
	}
	return userToResourceData(d, &u)
}
